# viagens.app

App privado de viagens para duas pessoas. Plano do produto em `docs/plano-app-viagens.md`; specs e planos em `docs/superpowers/`.

## Como trabalhar neste repositório

- Todo trabalho novo segue o Superpowers: brainstorming → spec em `docs/superpowers/specs/` → plano em `docs/superpowers/plans/` → TDD.
- Código segue o Ponytail (nível full): a menor mudança que resolve tudo, biblioteca padrão antes de dependência, sem opções ou abstrações que ninguém pediu. Limites conhecidos ficam marcados com `// shortcut: <limite>, <quando mudar>`.
- Nenhuma API paga (decisão do Gustavo): mapas e busca usam OpenStreetMap (OpenFreeMap, Nominatim). O mapa do Google é opcional, ligado no build (`VITE_MAP_PROVIDER=google`, ver `deploy/README.md`).
- Interface em português, mobile-first. Código, nomes e commits técnicos podem ser em inglês; mensagens ao usuário sempre em português.

## Comandos

- Backend: `cd backend && go vet ./... && go test ./...`
- Front: `cd frontend && npm test && npm run build` (o build vai para `backend/web/dist`, embutido no binário)
- Rodar local: ver "Rodar localmente" em `deploy/README.md`

## Estrutura

- `backend/` é um único `package main`: `auth.go` (rotas, sessão, CSRF), `trips.go`, `items.go`, `attachments.go`, `geocode.go`, `spa.go`, migrações em `migrations/`.
- `frontend/src/`: `api.ts` (fetch com header CSRF e redirect no 401), `days.ts` (monta os dias do roteiro), `pages/`, `components/TripMap.tsx`.
