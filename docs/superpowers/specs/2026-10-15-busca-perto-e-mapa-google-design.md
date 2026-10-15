# Busca por perto e mapa do Google

## Problema

1. "Restaurantes perto" e "Banheiros perto" sempre mostravam "Busca por perto indisponível agora". O backend mandava a consulta ao Overpass num POST com a consulta na URL e corpo vazio; o Overpass só lê o corpo num POST, recebia uma consulta vazia e respondia erro.
2. O Gustavo quer poder usar o mapa do Google, com uma chave no código que alterna entre Google e OpenStreetMap.

## Decisões

- Busca: a consulta vai no corpo (`application/x-www-form-urlencoded`). O teste falso do Overpass passa a ler só o corpo, como o real.
- Mapa: continua MapLibre. Com Google ligado, a fonte do mapa vira raster da **Map Tiles API** (2D tiles), carregada por um protocolo `gmaps://` que cria uma sessão por carregamento de página. Pins, bolinhas, linha do roteiro e busca não mudam.
- Toggle no build: `VITE_MAP_PROVIDER=google` + `VITE_GOOGLE_MAPS_KEY`. Sem os dois, OpenStreetMap (padrão). No Docker, vêm de `deploy/.env` como build args.
- Não usamos a Maps JavaScript API: exigiria reescrever pins, agrupamento e linha só para o Google.

## Plano (TDD)

1. Teste: o Overpass falso lê `r.PostForm` → falha; mandar a consulta no corpo → passa.
2. Teste `mapStyle.test.ts`: `mapProvider` escolhe Google só com chave; `googleTile` cria uma sessão, reaproveita, tenta de novo após falha e monta a URL do tile → implementar `mapStyle.ts`.
3. `TripMap` usa `STYLE` e registra o protocolo quando for Google. Conferido no navegador com tiles falsos.
4. Dockerfile/compose recebem as variáveis; `deploy/README.md` explica como configurar o Google.
