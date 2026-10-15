# Plano: correções da revisão geral

Spec: `docs/superpowers/specs/2026-10-15-revisao-geral-design.md`. TDD: cada item ganha um teste que falha antes da correção.

1. Front: `outbox.test.ts` (ids negativos, edição depois do envio, `item_id` entre reconexões, `reconcile`) → `outbox.ts`, `api.ts`, redirect em `ExpenseForm.tsx`/`ItemSheet.tsx`, deps do `OfflineBar.tsx`.
2. Backend: `fixes_test.go` + caso NaN em `nearby_test.go` → `auth.go` (limiter `try`/`ok`, bcrypt fictício), `items.go`, `expenses.go`, `rates.go` (memo), `nearby.go`, `checklist.go`, `attachments.go`.
3. `go vet ./... && go test ./...`, `npm test && npm run build`.
