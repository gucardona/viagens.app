# Etapa 4: gastos, checklist e offline — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** gastos com cotação e saldo entre os dois, checklist por viagem e viagem disponível offline.

**Architecture:** duas tabelas novas e um cache de cotações no SQLite; resumo calculado no backend; front ganha aba Gastos e aba Mais (Documentos + Checklist); um `sw.js` escrito à mão faz cache do app, da API (rede primeiro) e dos anexos.

**Tech Stack:** Go stdlib; React; Vitest. Sem dependências novas.

**Spec:** `docs/superpowers/specs/2026-10-13-etapa-4-gastos-checklist-offline-design.md`. Execução: nativa.

## Global Constraints

- Frankfurter: `GET https://api.frankfurter.app/{YYYY-MM-DD}?from={CUR}&to=BRL` → `{"rates":{"BRL":5.91}}`; data futura vira `latest`.
- Categorias: `lodging, transport, food, activity, shopping, other`.
- Valores em centavos (`amount_cents` > 0, até 10^11); moeda `^[A-Z]{3}$`; `payer_share` 0–100.
- Mensagens em português; sem API paga.

## Review Focus

1. Gasto em moeda sem cotação (Frankfurter fora ou moeda não coberta) e sem `rate`: fica fora do total e marcado, não quebra o resumo. → teste Task 2.
2. Saldo quando cada um pagou coisas diferentes, nos dois sentidos, e zero. → teste Task 2.
3. Apagar viagem apaga gastos e checklist (cascade). → teste Task 1/3.
4. Sem internet: abrir viagem guardada e um PDF guardado. → navegador, Task 5.
5. Deploy novo com service worker antigo: o app atualiza (cache do app versionado pelo build, `skipWaiting`). → navegador, Task 5.

---

### Task 1: gastos (CRUD) e `GET /api/users`

**Files:** Create `backend/migrations/004_expenses_checklist.sql`, `backend/expenses.go`, `backend/expenses_test.go`; Modify `backend/auth.go` (rotas, `ratesURL`).

**Produces:** `Expense{ID, TripID, ItemID *int64, Title, AmountCents int64, Currency, Category string, PaidBy int64, PayerShare int, Date string, Rate *float64, CreatedAt}`; JSON snake_case.

- [ ] Testes: criar/editar/apagar; validação (valor 0, moeda "eu", categoria x, share 101, paid_by inexistente, item de outra viagem → 400); `GET /api/users` devolve `[{id,name}]`; apagar viagem apaga gastos.
- [ ] Implementar; `go test ./...`; commit.

### Task 2: cotação e resumo

**Files:** Create `backend/rates.go`; Modify `backend/expenses.go`, `backend/expenses_test.go`.

**Produces:** `GET /api/trips/{id}/expenses` → `{expenses: [...+ brl_cents *int64], total_cents, by_category: {cat: cents}, by_day: {date: cents}, missing_rate: n, balance: {from, to, cents}|null}`; `func (a *App) rate(ctx, date, cur string) (float64, error)` com cache em `rates`.

- [ ] Testes: EUR busca cotação uma vez (segundo resumo não chama o Frankfurter falso); `rate` manual vence; BRL = 1; Frankfurter 500 → gasto com `brl_cents: null`, `missing_rate: 1`, fora do total; saldo: A paga 100 metade, B paga 40 metade → B deve 30 a A; A paga 100 com share 100 → saldo nulo.
- [ ] Implementar; commit.

### Task 3: checklist

**Files:** Create `backend/checklist.go`, `backend/checklist_test.go`.

- [ ] Testes: adicionar, marcar, apagar, listar em ordem; texto vazio → 400; copiar de outra viagem (textos desmarcados, sem duplicar o que já existe).
- [ ] Implementar; commit.

### Task 4: telas de gastos e checklist

**Files:** Create `frontend/src/money.ts`, `money.test.ts`, `pages/ExpensesTab.tsx`, `pages/ExpenseForm.tsx`, `pages/MoreTab.tsx`, `components/Checklist.tsx`; Modify `types.ts`, `main.tsx` (rota `/viagens/:id/gastos/:expenseId`), `pages/Trip.tsx` (abas Roteiro, Salvos, Mapa, Gastos, Mais), `pages/ItemSheet.tsx` ("Adicionar gasto", total do item).

**Produces:** `money(cents: number, cur = 'BRL'): string`; `parseMoney(s: string): number | null` ("12,50" → 1250); `categoryFor(kind: Kind): Category`.

- [ ] Testes Vitest de `money`, `parseMoney`, `categoryFor`.
- [ ] Telas; `npm test && npm run build`; commit.

### Task 5: offline

**Files:** Create `frontend/public/sw.js`, `frontend/src/offline.ts`; Modify `main.tsx` (registrar SW em produção), `pages/Trip.tsx` (botão e faixa), `backend/spa.go` (`sw.js` sem cache HTTP).

- [ ] `sw.js`: app shell (`/`, assets do build) cache-first com nome de cache por versão; `GET /api/*` rede primeiro com cópia; `/api/attachments/*` cache-first; navegação sem rede → `index.html` guardado.
- [ ] `offline.ts`: `saveTripOffline(id): Promise<{files, bytes}>` busca viagem, gastos, checklist e anexos; guarda data em `localStorage`.
- [ ] Navegador: baixar, ficar offline, abrir roteiro e PDF; commit.

### Task 6: PR
