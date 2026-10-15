# Plano: estatísticas

Spec: `docs/superpowers/specs/2026-10-15-estatisticas-design.md`.

1. `frontend/src/stats.test.ts` → `stats.ts` (feitas, planejadas, dias até hoje, países visitados vs. planejados com contagem).
2. `backend/pins_test.go` → `GET /api/pins` (só itens com pino, de todas as viagens; 401 sem sessão).
3. `pages/Stats.tsx` + rota + link na lista de viagens.
4. `npm test && npm run build`, `go vet && go test`, conferir no navegador, PR.
