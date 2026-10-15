# Estatísticas das viagens

Último item da V2 do plano ("Estatísticas: países visitados, mapa de todas as viagens").

## O que o usuário vê

Na lista de viagens, a linha de países vira um link "Ver estatísticas ›" para `/estatisticas`:

- Números: viagens feitas (já começaram), viagens planejadas, dias viajados (até hoje), países visitados.
- Países visitados: bandeira + nome em português + em quantas viagens. Países só de viagens futuras aparecem à parte, como "Planejados".
- Mapa de todas as viagens: cada lugar com pino, uma cor por viagem, agrupado em bolinhas quando perto; legenda com as viagens e suas cores. Tocar num pino abre o item.

## Como

- Países e datas já vêm de `GET /api/trips` (`countries` por viagem). A conta é uma função pura `stats(trips, today)` em `frontend/src/stats.ts`, com testes.
- Novo `GET /api/pins`: todos os itens com pino, de todas as viagens (mesmos campos de item), para o mapa.
- Cores: paleta fixa de 8 cores, por ordem da viagem na lista.
  `// shortcut: com mais de 8 viagens as cores se repetem; a legenda resolve.`
- "Hoje" é a data local do aparelho.
