package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	assistantModel = "claude-sonnet-5-5"
	maxToolRounds  = 10
	maxReadBytes   = 10 << 20
)

func (a *App) handleGetAssistantKey(w http.ResponseWriter, r *http.Request) {
	var key string
	if err := a.db.QueryRow(`SELECT anthropic_key FROM users WHERE id = ?`, userID(r)).Scan(&key); err != nil {
		serverError(w, err)
		return
	}
	hint := ""
	if len(key) >= 4 {
		hint = key[len(key)-4:]
	}
	writeJSON(w, 200, map[string]any{"set": key != "", "hint": hint})
}

// shortcut: the key is stored in plain text in the database (and its backups); encrypt it if the database ever leaves the VPS.
func (a *App) handlePutAssistantKey(w http.ResponseWriter, r *http.Request) {
	var in struct{ Key string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Key = strings.TrimSpace(in.Key)
	if len(in.Key) > 300 || strings.ContainsAny(in.Key, " \t\r\n") {
		writeError(w, 400, "Chave inválida")
		return
	}
	if _, err := a.db.Exec(`UPDATE users SET anthropic_key = ? WHERE id = ?`, in.Key, userID(r)); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

type chatMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type chatLink struct {
	Label string `json:"label"`
	To    string `json:"to"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// assistantTools call the app's own routes, so validation and limits stay in one place.
const assistantTools = `[
{"name":"get_trip","description":"Lê a viagem: itens do roteiro (com endereço, coordenadas, datas, horários, código de confirmação, observações, ingresso) e a lista de documentos anexados (id, nome, item_id). Itens sem data são lugares salvos.","input_schema":{"type":"object","properties":{}}},
{"name":"get_expenses","description":"Lê os gastos da viagem, os totais em reais por categoria e por dia, e quem deve a quem.","input_schema":{"type":"object","properties":{}}},
{"name":"search_nearby","description":"Busca no OpenStreetMap lugares perto de um ponto, ordenados por distância em metros: restaurant (restaurantes, cafés, lanchonetes, 800 m), attraction (museus, galerias, mirantes, atrações turísticas, 1,5 km) ou toilets (banheiros públicos, 800 m).","input_schema":{"type":"object","properties":{"what":{"type":"string","enum":["restaurant","attraction","toilets"]},"lat":{"type":"number"},"lng":{"type":"number"}},"required":["what","lat","lng"]}},
{"name":"search_place","description":"Procura um lugar ou endereço e devolve até 6 resultados com endereço e coordenadas. Os resultados preferem a região da localização atual.","input_schema":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}},
{"name":"add_item","description":"Adiciona um item ao roteiro desta viagem. Datas AAAA-MM-DD, horas HH:MM (hora exige data). Sem data, o item fica em Salvos. Inclua address, lat e lng quando souber o lugar (use search_place).","input_schema":{"type":"object","properties":{"kind":{"type":"string","enum":["flight","lodging","car","transport","restaurant","activity","event","note"]},"title":{"type":"string"},"start_date":{"type":"string"},"start_time":{"type":"string"},"end_date":{"type":"string"},"end_time":{"type":"string"},"address":{"type":"string"},"lat":{"type":"number"},"lng":{"type":"number"},"confirmation_code":{"type":"string"},"notes":{"type":"string"},"ticket":{"type":"string","enum":["","needed","bought"],"description":"needed = precisa comprar ingresso; bought = já comprado"},"time_approx":{"type":"boolean","description":"horário aproximado"}},"required":["kind","title"],"additionalProperties":false}},
{"name":"add_expense","description":"Adiciona um gasto. amount_cents é o valor em centavos na moeda (45 euros = 4500, EUR). paid_by é o id de quem pagou; sem ele, quem está falando. payer_share é a parte de quem pagou, em %: 50 divide meio a meio, 100 é só dele.","input_schema":{"type":"object","properties":{"title":{"type":"string"},"amount_cents":{"type":"integer"},"currency":{"type":"string","description":"código ISO de 3 letras"},"category":{"type":"string","enum":["lodging","transport","food","activity","shopping","other"]},"date":{"type":"string","description":"AAAA-MM-DD"},"paid_by":{"type":"integer"},"payer_share":{"type":"integer"},"item_id":{"type":"integer","description":"item do roteiro ligado ao gasto"}},"required":["title","amount_cents","currency","category","date"],"additionalProperties":false}},
{"name":"show_documents","description":"Mostra documentos anexados na conversa, com botão para abrir. Use quando a pessoa pedir um documento, ingresso, reserva ou comprovante.","input_schema":{"type":"object","properties":{"ids":{"type":"array","items":{"type":"integer"}}},"required":["ids"]}},
{"name":"read_document","description":"Lê o conteúdo de um documento anexado (PDF, imagem ou texto) para responder perguntas sobre ele, como código de reserva ou horário.","input_schema":{"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}}
]`

var assistantClient = &http.Client{Timeout: 3 * time.Minute}

type assistantError struct{ msg string }

func (e assistantError) Error() string { return e.msg }

// callClaude sends one Messages API request and returns the raw content and stop reason.
func (a *App) callClaude(ctx context.Context, key, system string, messages []any) (json.RawMessage, string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":         assistantModel,
		"max_tokens":    16000,
		"system":        system,
		"messages":      messages,
		"tools":         json.RawMessage(assistantTools),
		"output_config": map[string]string{"effort": "medium"},
		"fallbacks":     "default", // another model answers if a safety classifier declines
		"cache_control": map[string]string{"type": "ephemeral"},
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", a.anthropicURL, bytes.NewReader(body))
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "server-side-fallback-2026-07-01")
	req.Header.Set("content-type", "application/json")
	res, err := assistantClient.Do(req)
	if err != nil {
		slog.Warn("claude", "err", err)
		return nil, "", assistantError{"O Claude não respondeu. Tente de novo."}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != 200 {
		slog.Warn("claude", "status", res.StatusCode, "body", string(raw))
		switch {
		case res.StatusCode == 401 || res.StatusCode == 403:
			return nil, "", assistantError{"Chave da API do Claude inválida. Confira em “Trocar chave”."}
		case res.StatusCode == 429:
			return nil, "", assistantError{"Limite de uso da API do Claude atingido. Tente de novo em instantes."}
		case bytes.Contains(raw, []byte("credit balance")):
			return nil, "", assistantError{"A conta da API do Claude está sem créditos."}
		case res.StatusCode == 400:
			return nil, "", assistantError{"O Claude não aceitou o pedido. Tente uma nova conversa."}
		}
		return nil, "", assistantError{"O Claude está indisponível agora. Tente de novo."}
	}
	var out struct {
		StopReason string          `json:"stop_reason"`
		Content    json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", assistantError{"Resposta inesperada do Claude."}
	}
	return out.Content, out.StopReason, nil
}

// handleAssistant runs the tool loop for one chat message. api is the app's own router, used by the tools.
func (a *App) handleAssistant(api http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID, ok := pathID(w, r)
		if !ok {
			return
		}
		trip, ok := a.getTrip(w, tripID)
		if !ok {
			return
		}
		var in struct {
			Messages []chatMessage `json:"messages"`
			Lat, Lng *float64
		}
		if !readJSON(w, r, &in) {
			return
		}
		n := len(in.Messages)
		if n == 0 || n > 40 || in.Messages[n-1].Role != "user" {
			writeError(w, 400, "Conversa inválida")
			return
		}
		// shortcut: earlier turns come back as plain text, so what tools read in them is gone; Claude reads it again if needed.
		var messages []any
		for _, m := range in.Messages {
			if (m.Role != "user" && m.Role != "assistant") || strings.TrimSpace(m.Text) == "" || len([]rune(m.Text)) > 4000 {
				writeError(w, 400, "Conversa inválida")
				return
			}
			messages = append(messages, map[string]string{"role": m.Role, "content": m.Text})
		}

		var key, me string
		if err := a.db.QueryRow(`SELECT anthropic_key, name FROM users WHERE id = ?`, userID(r)).Scan(&key, &me); err != nil {
			serverError(w, err)
			return
		}
		if key == "" {
			writeError(w, 400, "Conecte sua chave da API do Claude para usar o chat")
			return
		}
		system, err := a.assistantSystem(trip, userID(r), me, in.Lat, in.Lng)
		if err != nil {
			serverError(w, err)
			return
		}

		t := &tools{a: a, api: api, r: r, tripID: tripID, lat: in.Lat, lng: in.Lng}
		var reply string
		for round := 0; ; round++ {
			content, stop, err := a.callClaude(r.Context(), key, system, messages)
			if err != nil {
				var ae assistantError
				if errors.As(err, &ae) {
					writeError(w, 502, ae.msg)
				} else {
					serverError(w, err)
				}
				return
			}
			var blocks []contentBlock
			json.Unmarshal(content, &blocks)
			reply = ""
			var results []any
			for _, b := range blocks {
				switch b.Type {
				case "text":
					reply += b.Text
				case "tool_use":
					results = append(results, t.run(b))
				}
			}
			if stop == "refusal" && reply == "" {
				reply = "Não posso ajudar com esse pedido."
			}
			if stop != "tool_use" || len(results) == 0 {
				break
			}
			if round == maxToolRounds {
				reply = "Parei aqui porque o pedido precisou de passos demais. Pode pedir de novo em partes?"
				break
			}
			// The assistant turn goes back unchanged (thinking blocks included), then all tool results in one message.
			messages = append(messages, map[string]any{"role": "assistant", "content": content},
				map[string]any{"role": "user", "content": results})
		}
		writeJSON(w, 200, map[string]any{"reply": strings.TrimSpace(reply), "links": t.links, "docs": t.docs})
	}
}

func (a *App) assistantSystem(trip Trip, me int64, myName string, lat, lng *float64) (string, error) {
	rows, err := a.db.Query(`SELECT id, name FROM users ORDER BY id`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var people []string
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return "", err
		}
		people = append(people, fmt.Sprintf("%s (id %d)", name, id))
	}
	var b strings.Builder
	fmt.Fprintf(&b, `Você é o consultor de viagens do app de viagens de um casal. Ajuda com dicas de lugares e atrações, endereços, documentos e mantém o roteiro e os gastos em dia.

Hoje é %s. Quem está falando: %s (id %d). Contas do app: %s.
Viagem: %q, destino %q, de %s a %s.
`, time.Now().Format(time.DateOnly), myName, me, strings.Join(people, ", "), trip.Name, trip.Destination, trip.StartDate, trip.EndDate)
	if lat != nil && lng != nil {
		fmt.Fprintf(&b, "Localização atual de quem está falando: %.5f, %.5f.\n", *lat, *lng)
	} else {
		b.WriteString("A localização atual não está disponível; para \"perto de mim\", pergunte onde a pessoa está ou use o hotel do dia.\n")
	}
	b.WriteString(`
Como responder:
- Português do Brasil, curto e direto, como numa conversa de celular. Texto simples, sem Markdown (sem asteriscos, títulos ou tabelas); listas com hífen.
- A pessoa pode estar falando por voz: interprete números e nomes com tolerância.
- Consulte get_trip antes de falar do roteiro, de endereços ou de documentos. Não invente o que está no roteiro.
- Para dicas perto, use search_nearby e diga a distância a pé. Combine com o que você sabe dos lugares, sem afirmar horários ou preços como certos.
- Ao adicionar um lugar ao roteiro, procure o endereço e as coordenadas com search_place. Se faltar algo essencial (dia de um gasto, valor), pergunte antes.
- Depois de adicionar algo, diga em uma linha o que foi adicionado; o app mostra o botão para abrir.
- Quando pedirem um documento, use show_documents. Use read_document para ler o conteúdo.
`)
	return b.String(), nil
}

// tools runs tool calls for one chat request and collects what the reply shows as buttons.
type tools struct {
	a        *App
	api      http.Handler
	r        *http.Request
	tripID   int64
	lat, lng *float64
	links    []chatLink
	docs     []Attachment
}

func toolResult(id string, isError bool, content any) map[string]any {
	return map[string]any{"type": "tool_result", "tool_use_id": id, "is_error": isError, "content": content}
}

// call sends a request to the app's own router with the caller's session.
func (t *tools) call(method, path string, body []byte) (int, []byte) {
	req, _ := http.NewRequestWithContext(t.r.Context(), method, path, bytes.NewReader(body))
	req.Header.Set("Cookie", t.r.Header.Get("Cookie"))
	rec := httptest.NewRecorder()
	t.api.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func (t *tools) run(b contentBlock) map[string]any {
	var in struct {
		What, Query string
		Lat, Lng    float64
		ID          int64
		IDs         []int64
	}
	json.Unmarshal(b.Input, &in)
	f := func(v float64) string { return fmt.Sprint(v) }
	trip := fmt.Sprint(t.tripID)
	var status int
	var out []byte
	switch b.Name {
	case "get_trip":
		status, out = t.call("GET", "/api/trips/"+trip, nil)
	case "get_expenses":
		status, out = t.call("GET", "/api/trips/"+trip+"/expenses", nil)
	case "search_nearby":
		status, out = t.call("GET", "/api/nearby?"+url.Values{"what": {in.What}, "lat": {f(in.Lat)}, "lng": {f(in.Lng)}}.Encode(), nil)
	case "search_place":
		q := url.Values{"q": {in.Query}}
		if t.lat != nil && t.lng != nil {
			q.Set("lat", f(*t.lat))
			q.Set("lng", f(*t.lng))
		}
		status, out = t.call("GET", "/api/geocode?"+q.Encode(), nil)
	case "add_item":
		status, out = t.call("POST", "/api/trips/"+trip+"/items", b.Input)
		var i Item
		if status == 201 && json.Unmarshal(out, &i) == nil {
			t.links = append(t.links, chatLink{"Roteiro: " + i.Title, fmt.Sprintf("/viagens/%d/itens/%d", t.tripID, i.ID)})
		}
	case "add_expense":
		var e map[string]any
		json.Unmarshal(b.Input, &e)
		if e == nil {
			e = map[string]any{}
		}
		if _, ok := e["paid_by"]; !ok {
			e["paid_by"] = userID(t.r)
		}
		body, _ := json.Marshal(e)
		status, out = t.call("POST", "/api/trips/"+trip+"/expenses", body)
		var ex Expense
		if status == 201 && json.Unmarshal(out, &ex) == nil {
			t.links = append(t.links, chatLink{"Gasto: " + ex.Title, fmt.Sprintf("/viagens/%d/gastos/%d", t.tripID, ex.ID)})
		}
	case "show_documents":
		var names []string
		for _, id := range in.IDs {
			at, err := t.a.getAttachment(id)
			if err != nil || at.TripID != t.tripID {
				return toolResult(b.ID, true, fmt.Sprintf("Documento %d não existe nesta viagem", id))
			}
			t.docs = append(t.docs, at)
			names = append(names, at.Filename)
		}
		return toolResult(b.ID, false, "Mostrando: "+strings.Join(names, ", "))
	case "read_document":
		return t.read(b.ID, in.ID)
	default:
		return toolResult(b.ID, true, "Ferramenta desconhecida")
	}
	if status >= 400 {
		var e struct{ Error string }
		json.Unmarshal(out, &e)
		return toolResult(b.ID, true, e.Error)
	}
	return toolResult(b.ID, false, string(out))
}

var imageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true}

func (t *tools) read(useID string, id int64) map[string]any {
	at, err := t.a.getAttachment(id)
	if err != nil || at.TripID != t.tripID {
		return toolResult(useID, true, "Documento não existe nesta viagem")
	}
	if at.Size > maxReadBytes {
		return toolResult(useID, true, "Documento grande demais para ler (mais de 10 MB); mostre com show_documents")
	}
	ctype := strings.TrimSpace(strings.Split(at.ContentType, ";")[0])
	var kind string
	switch {
	case ctype == "application/pdf":
		kind = "document"
	case imageTypes[ctype]:
		kind = "image"
	case ctype == "text/plain":
		kind = "text"
	default:
		return toolResult(useID, true, "Não consigo ler esse tipo de arquivo ("+ctype+"); mostre com show_documents")
	}
	data, err := os.ReadFile(t.a.filePath(id))
	if err != nil {
		return toolResult(useID, true, "Não consegui abrir o arquivo")
	}
	if kind == "text" {
		return toolResult(useID, false, at.Filename+":\n"+string(data))
	}
	return toolResult(useID, false, []any{map[string]any{"type": kind, "source": map[string]string{
		"type": "base64", "media_type": ctype, "data": base64.StdEncoding.EncodeToString(data)}}})
}
