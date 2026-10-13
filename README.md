# viagens.app

Organizador de viagens privado para duas pessoas: roteiro dia a dia, itens com endereço, pin no mapa e documentos anexados.

- `backend/`: Go (API + front embutido), SQLite
- `frontend/`: React + TypeScript (Vite), MapLibre com mapas OpenStreetMap
- `deploy/`: Docker Compose + Caddy. Guia em [deploy/README.md](deploy/README.md)
- `docs/`: plano do produto, specs e planos de implementação

Testes: `cd backend && go test ./...` e `cd frontend && npm test`.
