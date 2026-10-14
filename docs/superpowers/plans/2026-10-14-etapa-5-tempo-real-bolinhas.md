# Etapa 5: tempo real e bolinhas no mapa — plano

**Spec:** `docs/superpowers/specs/2026-10-14-etapa-5-tempo-real-bolinhas-design.md`. Execução: nativa.

### Task 1: `GET /api/events`
**Files:** `backend/events.go`, `backend/events_test.go`, `backend/auth.go` (rota, `events`, middleware `notifyChanges`).
- [x] Testes: duas conexões recebem `data: change` depois de um POST que deu certo; GET e 400 não avisam; sem sessão → 401.
- [x] Implementar; commit.

### Task 2: app busca de novo ao receber o aviso
**Files:** `frontend/src/live.ts`, `main.tsx`, `public/sw.js` (não interceptar `/api/events`), `pages/TripForm.tsx` (preencher uma vez só).
- [x] Navegador: duas contas, item criado por uma aparece na outra sem recarregar, com service worker ativo; commit.

### Task 3: bolinhas no mapa
**Files:** `frontend/src/cluster.ts` (`groupNear`), `cluster.test.ts`, `components/TripMap.tsx` (`cluster`), `pages/TripMapTab.tsx`, `pages/SavedTab.tsx`.
- [x] Vitest de `groupNear`; navegador: bolinha com contagem, toque aproxima e separa; commit.
