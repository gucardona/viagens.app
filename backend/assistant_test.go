package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClaude answers each Messages API call with the next reply and records the requests.
func fakeClaude(t *testing.T, app *App, replies ...string) *[]map[string]any {
	var reqs []map[string]any
	app.anthropicURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		body["_key"] = r.Header.Get("x-api-key")
		reqs = append(reqs, body)
		if len(reqs) > len(replies) {
			t.Errorf("unexpected call %d", len(reqs))
			w.WriteHeader(500)
			return
		}
		reply := replies[len(reqs)-1]
		if strings.HasPrefix(reply, "status:") {
			var code int
			fmt.Sscanf(reply, "status:%d", &code)
			w.WriteHeader(code)
			w.Write([]byte(`{"type":"error","error":{"type":"x","message":"x"}}`))
			return
		}
		w.Write([]byte(reply))
	})
	return &reqs
}

func toolUse(id, name, input string) string {
	return `{"stop_reason":"tool_use","content":[{"type":"text","text":"Vou ver."},{"type":"tool_use","id":"` + id + `","name":"` + name + `","input":` + input + `}]}`
}

func text(s string) string {
	return `{"stop_reason":"end_turn","content":[{"type":"text","text":"` + s + `"}]}`
}

func setKey(t *testing.T, c *http.Client, url, key string) {
	t.Helper()
	if res := do(t, c, "PUT", url+"/api/assistant/key", map[string]string{"key": key}); res.StatusCode != 204 {
		t.Fatalf("set key: %d", res.StatusCode)
	}
}

type chatReply struct {
	Reply string                       `json:"reply"`
	Links []struct{ Label, To string } `json:"links"`
	Docs  []Attachment                 `json:"docs"`
	Error string                       `json:"error"`
}

func chat(t *testing.T, c *http.Client, url string, tripID int64, msg string) (*http.Response, chatReply) {
	t.Helper()
	res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/assistant", url, tripID), map[string]any{
		"messages": []map[string]string{{"role": "user", "text": msg}}, "lat": 40.41, "lng": -3.70})
	return res, decode[chatReply](t, res)
}

func TestAssistantKey(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	got := decode[map[string]any](t, do(t, c, "GET", srv.URL+"/api/assistant/key", nil))
	if got["set"] != false {
		t.Fatalf("%v", got)
	}
	setKey(t, c, srv.URL, "  sk-ant-api03-segredo-abcd ")
	res := do(t, c, "GET", srv.URL+"/api/assistant/key", nil)
	var raw strings.Builder
	json.NewEncoder(&raw).Encode(decode[map[string]any](t, res))
	if !strings.Contains(raw.String(), `"set":true`) || !strings.Contains(raw.String(), `"hint":"abcd"`) || strings.Contains(raw.String(), "segredo") {
		t.Fatalf("%s", raw.String())
	}

	// The other account has its own (empty) key.
	createUser(app.db, "bruno@x.com", "Bruno", "segredo123")
	b := newClient()
	do(t, b, "POST", srv.URL+"/api/login", map[string]string{"email": "bruno@x.com", "password": "segredo123"})
	if got := decode[map[string]any](t, do(t, b, "GET", srv.URL+"/api/assistant/key", nil)); got["set"] != false {
		t.Fatalf("bruno sees %v", got)
	}

	setKey(t, c, srv.URL, "")
	if got := decode[map[string]any](t, do(t, c, "GET", srv.URL+"/api/assistant/key", nil)); got["set"] != false {
		t.Fatalf("not cleared: %v", got)
	}
}

func TestAssistantNeedsKey(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	res, got := chat(t, c, srv.URL, trip.ID, "oi")
	if res.StatusCode != 400 || !strings.Contains(got.Error, "chave") {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
}

func TestAssistantAddsItem(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	setKey(t, c, srv.URL, "sk-ant-teste")
	reqs := fakeClaude(t, app,
		toolUse("t1", "add_item", `{"kind":"activity","title":"Museu do Prado","start_date":"2026-11-03","start_time":"10:00","address":"Calle de Ruiz de Alarcón 23, Madrid"}`),
		text("Pronto, coloquei o Prado no dia 3."))
	res, got := chat(t, c, srv.URL, trip.ID, "coloca o Prado dia 3 às 10h")
	if res.StatusCode != 200 || got.Reply != "Pronto, coloquei o Prado no dia 3." {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	items, _ := app.listItems(trip.ID)
	if len(items) != 1 || items[0].Title != "Museu do Prado" || items[0].StartTime != "10:00" {
		t.Fatalf("%+v", items)
	}
	if len(got.Links) != 1 || got.Links[0].To != fmt.Sprintf("/viagens/%d/itens/%d", trip.ID, items[0].ID) || !strings.Contains(got.Links[0].Label, "Museu do Prado") {
		t.Fatalf("%+v", got.Links)
	}

	first, second := (*reqs)[0], (*reqs)[1]
	if first["_key"] != "sk-ant-teste" || first["model"] != "claude-sonnet-5-5" {
		t.Fatalf("%v %v", first["_key"], first["model"])
	}
	if sys := fmt.Sprint(first["system"]); !strings.Contains(sys, "40.41") || !strings.Contains(sys, trip.Name) || !strings.Contains(sys, "Ana") {
		t.Fatalf("system %s", sys)
	}
	// The second call carries the assistant turn back unchanged, then the tool result.
	msgs := second["messages"].([]any)
	if len(msgs) != 3 || msgs[1].(map[string]any)["role"] != "assistant" || !strings.Contains(fmt.Sprint(msgs[1]), "Vou ver.") || !strings.Contains(fmt.Sprint(msgs[1]), "t1") {
		t.Fatalf("%v", msgs)
	}
	result := msgs[2].(map[string]any)["content"].([]any)[0].(map[string]any)
	if result["type"] != "tool_result" || result["tool_use_id"] != "t1" || result["is_error"] == true || !strings.Contains(fmt.Sprint(result["content"]), "Museu do Prado") {
		t.Fatalf("%v", result)
	}
}

func TestAssistantToolErrorGoesBackToClaude(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	setKey(t, c, srv.URL, "sk-ant-teste")
	reqs := fakeClaude(t, app, toolUse("t1", "add_item", `{"kind":"bogus","title":"X"}`), text("Não deu."))
	if res, got := chat(t, c, srv.URL, trip.ID, "adiciona"); res.StatusCode != 200 || len(got.Links) != 0 {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	result := (*reqs)[1]["messages"].([]any)[2].(map[string]any)["content"].([]any)[0].(map[string]any)
	if result["is_error"] != true || !strings.Contains(fmt.Sprint(result["content"]), "Tipo inválido") {
		t.Fatalf("%v", result)
	}
}

func TestAssistantAddsExpensePaidBySpeaker(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	setKey(t, c, srv.URL, "sk-ant-teste")
	fakeClaude(t, app,
		toolUse("t1", "add_expense", `{"title":"Jantar","amount_cents":4500,"currency":"EUR","category":"food","date":"2026-11-03"}`),
		text("Anotado."))
	res, got := chat(t, c, srv.URL, trip.ID, "gastei 45 euros no jantar")
	if res.StatusCode != 200 || len(got.Links) != 1 || !strings.Contains(got.Links[0].To, "/gastos/") {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	var paidBy int64
	app.db.QueryRow(`SELECT paid_by FROM expenses`).Scan(&paidBy)
	var ana int64
	app.db.QueryRow(`SELECT id FROM users WHERE email = 'ana@x.com'`).Scan(&ana)
	if paidBy != ana {
		t.Fatalf("paid by %d, want %d", paidBy, ana)
	}
}

func TestAssistantReadsAndShowsDocument(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	setKey(t, c, srv.URL, "sk-ant-teste")
	pdf := []byte("%PDF-1.4 reserva ABC123")
	at := decode[Attachment](t, upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "hotel.pdf", "application/pdf", pdf, 0))
	reqs := fakeClaude(t, app,
		toolUse("t1", "read_document", fmt.Sprintf(`{"id":%d}`, at.ID)),
		toolUse("t2", "show_documents", fmt.Sprintf(`{"ids":[%d]}`, at.ID)),
		text("O código é ABC123."))
	res, got := chat(t, c, srv.URL, trip.ID, "qual o código do hotel?")
	if res.StatusCode != 200 || len(got.Docs) != 1 || got.Docs[0].ID != at.ID {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	result := (*reqs)[1]["messages"].([]any)[2].(map[string]any)["content"].([]any)[0].(map[string]any)
	block := result["content"].([]any)[0].(map[string]any)
	src := block["source"].(map[string]any)
	if block["type"] != "document" || src["media_type"] != "application/pdf" || src["data"] != base64.StdEncoding.EncodeToString(pdf) {
		t.Fatalf("%v", block)
	}
}

func TestAssistantBadKey(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	setKey(t, c, srv.URL, "sk-ant-errada")
	fakeClaude(t, app, "status:401")
	res, got := chat(t, c, srv.URL, trip.ID, "oi")
	if res.StatusCode != 502 || !strings.Contains(got.Error, "Chave") {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
}

func TestAssistantKeyIsEncryptedAtRest(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	setKey(t, c, srv.URL, "sk-ant-api03-segredo-abcd")
	var stored string
	app.db.QueryRow(`SELECT anthropic_key FROM users`).Scan(&stored)
	if strings.Contains(stored, "segredo") || !strings.HasPrefix(stored, "v1:") {
		t.Fatalf("stored %q", stored)
	}
	// The encryption key lives beside the database, not in it, readable only by the app.
	info, err := os.Stat(filepath.Join(app.dataDir, "secret.key"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("%v %v", info, err)
	}
	// A key saved before encryption is encrypted when the app starts.
	app.db.Exec(`UPDATE users SET anthropic_key = 'sk-ant-antiga-wxyz'`)
	again, err := newApp(app.dataDir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer again.db.Close()
	again.db.QueryRow(`SELECT anthropic_key FROM users`).Scan(&stored)
	if key, err := again.openKey(stored); strings.Contains(stored, "antiga") || err != nil || key != "sk-ant-antiga-wxyz" {
		t.Fatalf("stored %q key %q %v", stored, key, err)
	}
}
