# Plano: calendário .ics

Spec: `docs/superpowers/specs/2026-10-15-calendario-ics-design.md`.

1. `backend/calendar_test.go`: arquivo de uma viagem com item de dia inteiro, item com horário e fim, item sem data (ausente), escape, dobra de linha, 404 para viagem inexistente, 401 sem sessão. Ver falhar.
2. `backend/calendar.go`: `handleCalendar` + rota em `auth.go`. Ver passar; `go vet`.
3. Front: seção "Calendário" em `MoreTab.tsx` com link `download` para a rota. `npm test && npm run build`.
4. Conferir no navegador que o arquivo baixa; PR.
