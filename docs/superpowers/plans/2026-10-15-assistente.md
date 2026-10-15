# Plano: assistente de viagem

Spec: `docs/superpowers/specs/2026-10-15-assistente-design.md`.

1. `backend/assistant_test.go`: chave (grava, devolve só a dica, outra conta não vê, vazia apaga); chat sem chave dá 400; com uma API falsa: `add_item` cria o item e devolve o link, `read_document` manda o PDF como bloco `document`, `show_documents` devolve o anexo, 401 da API vira "chave inválida", x-api-key e modelo no pedido. Ver falhar.
2. `backend/migrations/006_assistant.sql` (005 está em uso no PR do link para a família), `backend/assistant.go`, rotas em `auth.go`, `attraction` em `nearby.go`. Ver passar; `go vet`.
3. Front: `pages/ChatTab.tsx` (chave, conversa, microfone, localização), aba em `Trip.tsx`, tipos. `npm test && npm run build`.
4. Conferir no navegador; PR.
