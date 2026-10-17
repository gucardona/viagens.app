# Plano: gastos por moeda

Spec: `docs/superpowers/specs/2026-10-17-gastos-por-moeda-design.md`.

1. Teste de `byCurrency` em `money.test.ts` (soma por moeda, real primeiro, depois em ordem alfabética) e ver falhar.
2. Implementar `byCurrency` e `moneyLine` em `money.ts`.
3. `ExpensesTab`: linha por moeda abaixo do total.
4. `ExpenseForm`: Moeda vira campo livre com sugestões; validar três letras antes de enviar.
5. `npm test && npm run build`, `go vet ./... && go test ./...`.
