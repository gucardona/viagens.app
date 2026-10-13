# Etapa 2: roteiro e mapa — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ingressos, horário aproximado e faixa de horário, país de cada item, mapa sempre visível e mapinha por dia.

**Architecture:** três colunas novas em `items` (+2 internas de geocodificação reversa), um processo em segundo plano no backend que preenche país e bairro pelo Nominatim reverso, e mudanças só de apresentação no front (funções puras testadas em `days.ts`/`geo.ts`).

**Tech Stack:** Go 1.26 stdlib + modernc sqlite; React 18 + maplibre-gl; Vitest.

**Spec:** `docs/superpowers/specs/2026-10-13-etapas-2-3-design.md` (seção "Etapa 2"). Execução: nativa (mesma escolha da etapa 1; o Gustavo aprovou seguir direto).

## Global Constraints

- Nenhuma API paga; Nominatim com `User-Agent: viagens.app (self-hosted)` e no máximo 1 requisição/s (o `geoGate` existente, compartilhado).
- Mensagens ao usuário em português; código em inglês.
- Sem dependências novas no front nem no back.
- `ticket` ∈ {`""`, `"needed"`, `"bought"`}; erro `"Ingresso inválido"`.

## Review Focus

1. Item sem pin ou pin no mar: o processo de países não pode ficar em laço nem travar; resposta 200 com `error` do Nominatim conta como consultado (país vazio). → teste em Task 2.
2. Nominatim fora do ar: o item salva normalmente e o país é preenchido numa rodada seguinte. → teste em Task 2.
3. Voo noturno (fim no dia seguinte) e hospedagem: o texto do horário não pode mostrar duração negativa nem faixa de vários dias para hotel. → testes em Task 3.
4. Viagem longa (15+ dias com pins): não pode criar 15 mapas WebGL de uma vez. → mapinha só monta perto da tela (Task 4), conferido no navegador.
5. Itens antigos (criados na etapa 1) sem as colunas novas: a migração usa valores padrão e o processo de países os preenche ao iniciar. → teste em Task 1/2.

---

### Task 1: migração e campos `ticket` / `time_approx`

**Files:** Create `backend/migrations/002_ticket_time_geo.sql`; Modify `backend/items.go`; Test `backend/items_test.go`.

**Interfaces — Produces:** `Item` ganha `Ticket string json:"ticket"`, `TimeApprox bool json:"time_approx"`, `CountryCode string json:"country_code"`, `Area string json:"area"`; `itemInput` ganha `Ticket`, `TimeApprox` (country/area não são aceitos na entrada).

- [ ] Teste `TestItemTicketAndApprox`: criar item com `ticket:"needed", time_approx:true` → 201 e os dois voltam; `PUT` com `ticket:"bought"` volta `bought`; `ticket:"x"` → 400 `"Ingresso inválido"`; item criado sem os campos volta `ticket:""`, `time_approx:false`, `country_code:""`.
- [ ] Rodar `go test ./... -run TestItemTicket` → falha.
- [ ] Migração: `ALTER TABLE items ADD COLUMN ticket TEXT NOT NULL DEFAULT ''`, idem `time_approx INTEGER NOT NULL DEFAULT 0`, `country_code TEXT NOT NULL DEFAULT ''`, `area TEXT NOT NULL DEFAULT ''`, `geo_key TEXT NOT NULL DEFAULT ''`. Atualizar `itemCols`, `scanItem`, INSERT e UPDATE; validação em `validate()`.
- [ ] `go vet ./... && go test ./...` → passa. Commit.

### Task 2: país e bairro em segundo plano

**Files:** Create `backend/places.go`, `backend/places_test.go`; Modify `backend/auth.go` (campos do App), `backend/main.go` (iniciar o laço), `backend/items.go` (avisar depois de salvar), `backend/trips.go` (`countries` na lista).

**Interfaces — Produces:** `func (a *App) fillGeo(ctx context.Context) error` (uma rodada; devolve o primeiro erro de rede para o laço registrar); `func (a *App) geoLoop(ctx context.Context)`; `func (a *App) kickGeo()` (envio não bloqueante num `chan struct{}` com buffer 1); `App.reverseURL string` (padrão `https://nominatim.openstreetmap.org/reverse`); `Trip.Countries []string json:"countries,omitempty"` preenchido só em `GET /api/trips`.

- [ ] Testes (Nominatim falso, `app.geoGate = &rateGate{}` para não esperar):
  - `TestFillGeo`: item com pin → depois de `fillGeo`, `country_code:"pt"`, `area:"Alfama"` (resposta `{"address":{"suburb":"Alfama","city":"Lisboa","country_code":"pt"}}`); a requisição tem `lat`, `lon`, `format=jsonv2`, `zoom=14`, `accept-language=pt-BR`; segunda rodada não faz requisição; mudar o pin faz consultar de novo.
  - `TestFillGeoFallbackArea`: sem `suburb`/`neighbourhood`/`city_district` usa `city`, depois `town`, depois `village`.
  - `TestFillGeoNoResult`: resposta 200 `{"error":"Unable to geocode"}` → `country_code:""` e não consulta de novo.
  - `TestFillGeoUpstreamDown`: 500 → `fillGeo` devolve erro, item fica sem país; com o servidor voltando, a rodada seguinte preenche.
  - `TestTripListCountries`: dois itens em `pt` e um em `es` → `countries:["es","pt"]` (ordenado, sem repetir).
- [ ] Rodar → falha.
- [ ] Implementar. `geo_key = fmt.Sprintf("%.5f,%.5f", lat, lng)`. Área: primeiro não vazio de `suburb, neighbourhood, city_district, city, town, village`. `geoLoop`: `fillGeo` e espera `kick` ou 10 min, até `ctx` acabar. `main.go` inicia `go app.geoLoop(context.Background())` só no `serve`. Create/Update chamam `a.kickGeo()`.
- [ ] `go vet ./... && go test ./...` → passa. Commit.

### Task 3: textos de horário, ingresso e bandeira (funções puras)

**Files:** Modify `frontend/src/types.ts`, `frontend/src/days.ts`, `frontend/src/days.test.ts`; Create `frontend/src/geo.ts`, `frontend/src/geo.test.ts`.

**Interfaces — Produces:** `timeLabel(i: Item, date: string): string`; `pinNumbers(items: Item[]): Map<number, number>` (id → 1..n na ordem dada, só itens com pin); `flag(code: string): string`; `Item` ganha `ticket: '' | 'needed' | 'bought'`, `time_approx: boolean`, `country_code: string`, `area: string`; `Trip` ganha `countries?: string[]`.

- [ ] Testes `timeLabel`:
  - `09:00`→`11:00` no mesmo dia → `"09:00–11:00 · 2h"`; `09:00`→`10:30` → `"09:00–10:30 · 1h30"`; `09:00`→`09:45` → `"09:00–09:45 · 45min"`.
  - só início → `"09:00"`; aproximado → `"~09:00"`; aproximado com fim → `"~09:00–11:00 · 2h"`.
  - voo `22:10` dia 3 → `07:30` dia 4 → `"22:10–07:30 (+1)"` (sem duração).
  - hospedagem: no check-in `"15:00"` (sem faixa), no check-out `"11:00"`.
  - sem hora → `""`.
- [ ] Teste `pinNumbers`: `[a(pin), b(sem pin), c(pin)]` → `{a:1, c:2}`.
- [ ] Teste `flag('pt')` → `"🇵🇹"`; `flag('')` → `""`.
- [ ] `npm test` → falha; implementar; `npm test` → passa. Commit.

### Task 4: telas

**Files:** Modify `frontend/src/components/TripMap.tsx`, `frontend/src/pages/Trip.tsx`, `frontend/src/pages/TripMapTab.tsx`, `frontend/src/pages/ItemSheet.tsx`, `frontend/src/pages/Trips.tsx`; Create `frontend/src/components/DayMap.tsx`.

**Interfaces — Consumes:** Task 3. **Produces:** `TripMap` ganha `numbers?: Map<number, number>` (pin mostra o número no lugar do emoji), `geolocate?: boolean` (adiciona `GeolocateControl`) e `interactive?: boolean` (padrão `true`; `false` para o mapinha).

- [ ] `TripMap`: monta sempre; com zero pins fica no mundo.
- [ ] `DayMap`: `IntersectionObserver` com `rootMargin: '300px'`; monta `TripMap` (`h-40`, `line`, `numbers`, `interactive={false}`) só enquanto visível; tocar leva a `?aba=mapa&dia=<data>`.
- [ ] Roteiro: cartão usa `timeLabel`, mostra o número do pin, selo `🎫 comprar` (amarelo) / `🎫 comprado` (verde); `DayMap` abaixo do título do dia quando há pin; bloco "N ingressos para comprar" no topo, abre a lista com data e link de cada item.
- [ ] Cabeçalho da viagem: bandeiras dos `country_code` dos itens.
- [ ] Aba Mapa: sempre mostra o mapa; faixa de ajuda quando não há pin; dia inicial lido de `?dia=`; `geolocate`.
- [ ] Item: escolha "Ingresso" (Não precisa / Precisa comprar / Comprado) e caixa "Horário aproximado".
- [ ] Lista de viagens: bandeiras no cartão e "N países" no topo quando > 0.
- [ ] `npm test && npm run build` → passa. Conferir no navegador em 390×844 com o backend local: criar item com ingresso a comprar e horário aproximado; ver selo, bloco do topo, mapinha numerado, aba Mapa vazia com mapa. Commit.

### Task 5: PR

- [ ] `cd backend && go vet ./... && go test ./...`; `cd frontend && npm test && npm run build`.
- [ ] Atualizar `docs/superpowers/specs/...` status para "aprovado em 13/10"; push; abrir PR "Etapa 2: roteiro e mapa".
