# Etapa 3: lugares salvos e o que tem por perto — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** aba Salvos com agrupamento por proximidade e "levar para um dia", importação do Takeout e de links do Google Maps, restaurantes e banheiros por perto.

**Architecture:** salvos são itens sem data. O backend ganha três rotas (`nearby` pelo Overpass, `maps-link`, `import`) e a busca por nome dos importados sem pin entra no processo de segundo plano da etapa 2. Agrupamento, leitura dos arquivos e ordem de caminhada ficam em funções puras no front.

**Tech Stack:** Go stdlib; React + maplibre-gl; Vitest. Sem dependências novas.

**Spec:** `docs/superpowers/specs/2026-10-13-etapas-2-3-design.md` (seção "Etapa 3"). Execução: nativa.

## Global Constraints

- Overpass `https://overpass-api.de/api/interpreter`, raio 800 m, até 40 resultados, cache 1 h por (what, lat/lng arredondados a 3 casas).
- Restaurantes: `amenity` ∈ restaurant, cafe, fast_food, com nome. Banheiros: `amenity=toilets` sem `access` private/no/customers.
- Links seguidos só para `maps.app.goo.gl`, `goo.gl`, `google.com`, `www.google.com`, `maps.google.com`, `consent.google.com`; no máximo 5 saltos.
- Duplicado na importação: mesmo título (sem diferenciar maiúsculas) e os dois sem pin, ou pins a menos de 50 m.
- "Levar para um dia": a partir de 10:00, 1h30 entre itens, `time_approx: true`; itens que passariam de 23:59 ficam sem hora.
- Agrupamento: DBSCAN, `minPts = 2`, raio 0,5–3 km (padrão 1 km).

## Review Focus

1. Link do Google que redireciona para um domínio fora da lista: recusado, sem fazer a requisição. → teste Task 2.
2. GeoJSON do Takeout com coordenadas `[0,0]` (lugar sem local): entra sem pin, não no Golfo da Guiné. → teste Task 4.
3. CSV com vírgula e aspas dentro do título/nota: lido certo. → teste Task 4.
4. Overpass fora do ar: 502 com mensagem; os salvos continuam aparecendo. → teste Task 1 e tela.
5. Grupo com mais de 9 lugares levado para um dia: horários não passam de 23:59. → teste Task 5.

---

### Task 1: `GET /api/nearby`

**Files:** Create `backend/nearby.go`, `backend/nearby_test.go`; Modify `backend/auth.go` (rota, `overpassURL`, cache).

**Produces:** `type Nearby struct { Name string; Lat, Lng float64; Distance int /*m*/; Kind, Cuisine, OpeningHours string; Fee, Wheelchair string }` com JSON `name, lat, lng, distance, kind, cuisine, opening_hours, fee, wheelchair`.

- [ ] Testes (Overpass falso): `TestNearbyRestaurants` (nó e `way` com `center` viram resultados ordenados por distância; a consulta enviada contém `around:800,38.7,-9.1` e `restaurant|cafe|fast_food`); `TestNearbyToilets` (sem nome vira "Banheiro público"; `fee`/`wheelchair` repassados); `TestNearbyCache` (segunda chamada igual não chega ao Overpass); `TestNearbyBadInput` (`what=x` ou lat inválida → 400); `TestNearbyUpstreamDown` (500 → 502 "Busca por perto indisponível agora").
- [ ] Implementar; `go test ./...`; commit.

### Task 2: `POST /api/maps-link`

**Files:** Create `backend/mapslink.go`, `backend/mapslink_test.go`.

**Produces:** `parseMapsURL(u *url.URL) (title string, lat, lng *float64)`; handler devolve `{title, lat, lng}`; `App.linkClient *http.Client` (substituível no teste).

- [ ] Testes de `parseMapsURL`: `/maps/place/Past%C3%A9is+de+Bel%C3%A9m/@38.69,-9.20,17z/data=!3m1!4b1!4m6!3m5!1s0x0:0x0!8m2!3d38.6975!4d-9.2032` → "Pastéis de Belém", 38.6975, -9.2032 (`!3d!4d` vence `@`); só `@` → usa `@`; `?q=38.7,-9.1` → coords; `?q=Torre+de+Bel%C3%A9m` → título sem coords; `?api=1&query=38.7,-9.1`.
- [ ] Testes do handler: link curto (servidor falso como `maps.app.goo.gl` via `linkClient` com transporte que redireciona) segue até `google.com/maps/...`; redirecionamento para `evil.com` → 400 "Link do Google Maps inválido"; URL inicial fora da lista → 400.
- [ ] Implementar; commit.

### Task 3: importação e busca por nome em segundo plano

**Files:** Create `backend/migrations/003_search_pending.sql`, `backend/import.go`, `backend/import_test.go`; Modify `backend/places.go`.

**Produces:** `POST /api/trips/{id}/import` com `[{kind?, title, notes, url, lat, lng}]` (máx. 2000) → `{created, skipped}`; `kind` padrão `activity`; sem lat/lng tenta `parseMapsURL(url)`; ainda sem pin → `search_pending = 1`. `fillGeo` procura antes os pendentes: Nominatim `search?q=<título>, <destino>&limit=1`; achou → grava pin (e `kind = restaurant` se `category=amenity` e `type` ∈ restaurant, cafe, fast_food, bar, pub); resposta 200 → `search_pending = 0`.

- [ ] Testes: cria itens sem data com pin do `lat/lng` e do `url`; duplicados ignorados (mesmo título sem pin; mesmo título a 30 m) e contados em `skipped`; título vazio → 400; busca por nome grava pin e vira restaurante; não achou → continua sem pin e não busca de novo.
- [ ] Implementar; commit.

### Task 4: leitura do Takeout e agrupamento (funções puras)

**Files:** Create `frontend/src/takeout.ts`, `frontend/src/takeout.test.ts`, `frontend/src/cluster.ts`, `frontend/src/cluster.test.ts`.

**Produces:** `type Place = { title: string; notes: string; url: string; lat: number | null; lng: number | null }`; `parseTakeout(name: string, text: string): Place[]`; `parseCSV(text: string): string[][]`; `distanceKm(a, b): number`; `cluster<T extends {lat,lng}>(points: T[], epsKm: number): { groups: T[][]; alone: T[] }`; `walkOrder<T>(points: T[]): T[]`; `dayTimes(n: number): string[]`.

- [ ] Testes: GeoJSON (nome em `properties.location.name` ou `properties.Title`; `[0,0]` → sem pin; nota em `properties.Comment`); CSV com aspas, vírgulas e linhas vazias; `cluster` (três pontos a 300 m + um a 5 km com eps 1 km → um grupo de 3 e 1 avulso; eps 0,1 → todos avulsos); `walkOrder` começa no mais a oeste e segue o vizinho mais próximo; `dayTimes(3)` → `['10:00','11:30','13:00']`, `dayTimes(11)[10]` → `''`.
- [ ] Implementar; commit.

### Task 5: telas

**Files:** Create `frontend/src/pages/SavedTab.tsx`, `frontend/src/components/Nearby.tsx`, `frontend/src/components/ImportDialog.tsx`; Modify `Trip.tsx`, `TripMapTab.tsx`, `ItemSheet.tsx`, `components/TripMap.tsx`.

**Produces:** `TripMap` ganha `colors?: Map<number, string>`, `extra?: { lat: number; lng: number; emoji: string; label: string }[]` e `onMove?: (c: {lat:number,lng:number}) => void`.

- [ ] Abas: Roteiro, Salvos, Mapa, Documentos. Roteiro sem "Sem data".
- [ ] Salvos: mapa com cor por grupo, controle de raio, grupos ("Alfama · 6 lugares"), Avulsos, Sem local; "Levar para um dia" (escolha do dia → `PUT` de cada item com data e horário de `dayTimes`); botões "Importar do Google" e "Colar link"; "+ Lugar" abre item novo sem data.
- [ ] Por perto: botões "🍽️ Restaurantes perto" e "🚻 Banheiros perto"; restaurantes salvos a até 1 km primeiro, depois os novos (sem repetir), "Salvar" via import; erro do Overpass em texto.
- [ ] Item com pin e aba Mapa (centro do mapa) usam `Nearby`.
- [ ] `npm test && npm run build`; navegador em 390×844 com Overpass falso. Commit.

### Task 6: PR

- [ ] Todos os testes; push depois do merge do PR #5 (rebase em `main`); PR "Etapa 3: lugares salvos e por perto".
