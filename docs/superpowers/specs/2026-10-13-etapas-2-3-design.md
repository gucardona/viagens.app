# Etapas 2 e 3: roteiro completo, mapa, lugares salvos e o que tem por perto (design)

Status: aprovado pelo Gustavo em 13/10
Plano geral: `docs/plano-app-viagens.md`. Base: etapa 1 (`2026-10-12-etapa-1-base-design.md`), já no ar.

## O que o Gustavo pediu (13/10)

> prossiga com o resto do projeto, vejo que o mapa não está lá, detecção de países, etc
> - datas e horários
> - o que tem que comprar ingresso
> - o que já tem ingresso comprado
> - sugestão de restaurantes perto (dentro daqueles que já salvamos e novos)
> - banheiros públicos
> - +- horário de cada coisa
> - um mapinha com cada coisa

Mais o que já estava no plano para a etapa 2: lugares salvos, importação do Google Takeout e agrupamento por proximidade.

## Como estou interpretando (corrija se estiver errado)

- **"O mapa não está lá"**: a aba Mapa hoje só mostra um texto quando nenhum item tem pin, e o Roteiro não tem mapa nenhum. Passa a ter mapa sempre.
- **"Detecção de países"**: o app descobre sozinho o país de cada item pelo pin, mostra as bandeiras na viagem e conta os países visitados.
- **"Datas e horários" e "+- horário de cada coisa"**: cada item mostra início e fim (`09:00–11:00 · 2h`) e pode ter horário aproximado (`~09:00`), para planejar sem fingir precisão.
- **"Um mapinha com cada coisa"**: cada dia do roteiro ganha um mapa pequeno com os itens do dia numerados na ordem da lista.
- **Restaurantes e banheiros**: busca no OpenStreetMap (gratuito) em volta de um item, do centro do mapa ou de onde vocês estão.
- **Lugares salvos** são itens sem data. Não crio uma tabela nova: o que já aparece em "Sem data" vira a lista "Quero ir", com agrupamento e importação.

## Divisão em duas etapas (um PR cada)

- **Etapa 2: roteiro e mapa.** Mapa sempre visível, mapinha por dia, horários, ingressos, países.
- **Etapa 3: lugares salvos e por perto.** Aba Salvos com agrupamento, importação do Takeout e de links do Google Maps, restaurantes e banheiros por perto.
- Gastos, checklist e documentos offline passam a ser a **etapa 4**.

---

## Etapa 2: roteiro e mapa

### Dados (migração `002`)

```
items + ticket       TEXT    ''  -- '' não precisa, 'needed' precisa comprar, 'bought' comprado
      + time_approx  INTEGER 0   -- 1 = horário aproximado
      + country_code TEXT    ''  -- ISO 3166-1 alfa-2 em minúsculas, preenchido pelo app
      + area         TEXT    ''  -- bairro ou cidade, preenchido pelo app (usado na etapa 3)
      + geo_key      TEXT    ''  -- "lat,lng" da última consulta; diferente do pin atual = consultar de novo
```

`ticket` e `time_approx` entram no formulário e na API (`PUT`/`POST` de itens); `ticket` só aceita os três valores. `country_code` e `area` são só leitura na API.

### Países

- Um processo em segundo plano no backend procura itens com pin cujo `geo_key` não bate com o pin atual, consulta o Nominatim reverso (`/reverse?zoom=14`, mesmo limite de 1 por segundo da busca) e grava `country_code` e `area` (bairro, ou cidade se não houver bairro).
- Ele roda ao iniciar o app (isso preenche os itens que já existem), a cada 10 minutos, e logo depois de salvar um item. Se o Nominatim falhar, tenta de novo na próxima rodada; nada trava o salvamento do item.
- **Viagem**: bandeiras dos países dos itens ao lado do destino. **Lista de viagens**: bandeiras em cada cartão e, no topo, "N países" contando todas as viagens.
- A bandeira é o emoji montado a partir do código (sem imagens nem biblioteca).

### Horários

- Cartão do roteiro: `09:00–11:00 · 2h`. Sem fim: só o início. Fim em outro dia (voo noturno): `22:10–07:30 (+1)`.
- Horário aproximado: caixa "Horário aproximado" no item; o roteiro mostra `~09:00` em cinza.
- Hospedagem continua mostrando check-in num dia e check-out no outro.

### Ingressos

- No item: escolha "Ingresso: não precisa / precisa comprar / comprado". Vale para qualquer tipo, mas o padrão é "não precisa".
- Roteiro: selo amarelo "🎫 comprar" ou verde "🎫 comprado" no cartão.
- Topo do roteiro: se houver ingressos para comprar, um bloco "3 ingressos para comprar" que abre a lista desses itens com a data de cada um. Marcar como comprado é editar o item; o comprovante vai como anexo, como já funciona.

### Mapas

- **Aba Mapa**: o mapa aparece sempre. Sem pins, mostra o mundo com uma faixa "Abra um item e toque no mapa para marcar onde ele fica". Botão "📍" do próprio MapLibre centraliza em onde vocês estão (permissão do navegador).
- **Mapinha por dia** no Roteiro: cada dia com pelo menos um pin mostra um mapa de 160 px com os pins numerados (1, 2, 3, na ordem da lista) e a linha ligando-os. O cartão do item mostra o mesmo número. Tocar no mapinha abre a aba Mapa já filtrada naquele dia.
- Navegadores limitam quantos mapas WebGL podem existir ao mesmo tempo (cerca de 16), então um mapinha só é montado quando o dia está perto da tela e é desmontado quando sai.

---

## Etapa 3: lugares salvos e o que tem por perto

### Aba "Salvos"

- Nova aba entre Roteiro e Mapa. Lista os itens sem data da viagem ("Quero ir").
- **Agrupamento por proximidade** (DBSCAN com distância real, em TypeScript no navegador, sobre algumas centenas de pontos): controle deslizante de 500 m a 3 km, padrão 1 km. Cada grupo tem o nome do bairro mais comum (`area`), a contagem e uma cor; os isolados ficam em "Avulsos"; os sem pin ficam em "Sem local", com atalho para marcar no mapa.
- Um mapa no topo mostra os salvos com a cor do grupo.
- **"Levar para um dia"** num grupo: escolhe o dia, e os itens recebem a data e horários aproximados a partir das 10:00, de 1h30 em 1h30, na ordem de menor caminhada (vizinho mais próximo, começando pelo mais a oeste). Tudo fica marcado como aproximado para vocês ajustarem.
- O Roteiro deixa de mostrar "Sem data"; esses itens estão na aba Salvos.

### Importar

- **Google Takeout**: na aba Salvos, "Importar do Google". Vocês escolhem um ou mais arquivos do Takeout já descompactado:
  - `Saved Places.json` / `Lugares com estrela` (GeoJSON): entram com pin.
  - Listas em `.csv` (Quero ir, Favoritos, listas próprias): título, nota e link. Se o link tiver coordenadas, entram com pin; se não, entram sem pin e o processo em segundo plano procura pelo nome junto com o destino da viagem no Nominatim. Se achar, grava o pin; se o resultado for um restaurante ou café, o tipo vira Restaurante. Os que não forem achados ficam em "Sem local".
  - Antes de gravar, uma tela mostra quantos lugares vieram de cada arquivo e quantos já existem na viagem (mesmo título e pin a menos de 50 m, ignorados).
- **Colar link do Google Maps**: "Colar link" cria um item salvo. O backend lê nome e coordenadas do link (`@lat,lng`, `!3d…!4d…`, `?q=lat,lng`). Links curtos (`maps.app.goo.gl`, `goo.gl/maps`) são seguidos só nesses domínios e só até o `google.com/maps`, para o servidor não virar porta para outros sites. Sem coordenadas no link, cai na busca por nome.
- Arquivo `.zip` direto não entra: abrir zip no navegador pede uma biblioteca, e descompactar no celular ou computador resolve.

### Por perto: restaurantes e banheiros

- `GET /api/nearby?lat=&lng=&what=restaurant|toilets` consulta o Overpass (OpenStreetMap, gratuito) num raio de 800 m (restaurantes: restaurante, café, fast food; banheiros: `amenity=toilets` sem acesso privado). Devolve nome, pin, distância, tipo de cozinha, horário de funcionamento e, nos banheiros, se é pago e se é acessível. Cache em memória de 1 hora por região (~100 m), no máximo 40 resultados, os mais próximos primeiro.
- **Na tela do item** (com pin) e **na aba Mapa** (em volta do centro do mapa, ou de onde vocês estão depois do "📍"): botões "🍽️ Restaurantes perto" e "🚻 Banheiros perto".
- Restaurantes: primeiro os **que vocês já salvaram** na viagem a até 1 km (com distância), depois os **novos** do OpenStreetMap, sem repetir os já salvos. Cada novo tem "Salvar", que cria um item Restaurante sem data. Banheiros não são salvos, só aparecem no mapa e na lista com distância e "Como chegar".
- Se o Overpass estiver fora, a lista mostra "Busca por perto indisponível agora" e os salvos continuam aparecendo.

### API nova (etapa 3)

```
GET  /api/nearby?lat=&lng=&what=restaurant|toilets
POST /api/maps-link                {url} -> {title, lat, lng}  (lat/lng nulos se o link não tiver)
POST /api/trips/{id}/import        [{title, notes, lat, lng}] -> {created, skipped}
```

O agrupamento e a leitura dos arquivos do Takeout ficam no navegador; o backend só recebe a lista final.

---

## Testes

- Go (`httptest`, Nominatim e Overpass falsos): migração 002; validação de `ticket`; processo de países (preenche, refaz quando o pin muda, tenta de novo depois de falha, respeita o limite); leitura de links do Google Maps (formatos acima, domínio não permitido recusado, redirecionamento para fora recusado); importação (duplicado ignorado, sem pin vai para a busca por nome); nearby (formato, cache, Overpass fora do ar).
- Vitest: texto do horário (`~`, faixa, duração, `+1`); números dos pins na ordem do dia; DBSCAN (grupos, avulsos, raio); ordem de menor caminhada e horários de "Levar para um dia"; leitura do GeoJSON e do CSV do Takeout.
- No navegador em tamanho de celular: ingresso para comprar aparece no topo; mapinha por dia; aba Mapa vazia mostra o mapa; importar um CSV de exemplo; agrupar; levar um grupo para um dia; restaurantes e banheiros com Overpass falso (este ambiente bloqueia o OpenStreetMap, então a busca real só dá para ver em produção).

## Riscos e limites

- Overpass e Nominatim são serviços públicos com limite de uso justo. Para duas pessoas está muito abaixo; o cache e o limite de 1 por segundo cobrem picos como a importação de 300 lugares (cerca de 5 minutos em segundo plano).
- OpenStreetMap tem menos restaurantes e horários que o Google, principalmente fora de capitais.
- Achar pelo nome um lugar de lista do Takeout erra às vezes (dois lugares com o mesmo nome na cidade). O pin aparece no mapa e pode ser movido com um toque.
- Ordem "de menor caminhada" é aproximada (vizinho mais próximo, em linha reta), não rota real.
