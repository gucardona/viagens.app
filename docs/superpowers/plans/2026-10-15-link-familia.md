# Plano: compartilhar viagem

Spec: `docs/superpowers/specs/2026-10-15-link-familia-design.md`.

1. `backend/share_test.go`: criar (idempotente), aparece na viagem, família lê sem sessão e sem campos privados, desativar → 404, viagem inexistente → 404, sem sessão → 401, token inválido → 404. Ver falhar.
2. Migração + `share.go` + rotas. Ver passar.
3. Front: `Shared.tsx` + rota, `Day`/`DayMap` em modo leitura, `live.ts` desligado no link, seção em `MoreTab.tsx`.
4. Navegador: criar link, abrir num contexto sem cookie, conferir que não vaza nada e que desativar funciona.
