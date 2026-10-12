# Organizador de viagens: proposta de plano

App web privado para vocês dois, que funciona bem no celular (instalável como app pela tela inicial) e no computador.

> **Atualização (12/10):** sem nenhuma API paga do Google. Busca e importação usam OpenStreetMap (Nominatim/Photon); lugares com estrela do Takeout entram com coordenadas; listas salvas são procuradas pelo nome e as não encontradas são marcadas no mapa; links compartilhados do Google Maps têm as coordenadas lidas do próprio link.

## 1. Escopo da primeira versão (V1)

**Viagens**
- Criar viagem com nome, datas, destino(s), capa e participantes (vocês dois).
- Linha do tempo dia a dia, no estilo Tripsy: cada dia lista os itens em ordem, com horário.

**Itens do roteiro** (todos com endereço, pin no mapa, notas, anexos e links)
- Tipos: voo, hospedagem, aluguel de carro, trem/ônibus, restaurante, atração/passeio, evento, nota livre.
- Campos de reserva: código de confirmação, check-in/out, horários, contato, valor pago e moeda.
- Anexos: PDF, foto, comprovante (passagem, voucher, seguro, passaporte) guardados no app e abertos com um toque.
- Botão "Abrir no Google Maps / Waze" e "Como chegar" em cada item.

**Mapa**
- Mapa da viagem com todos os pins, coloridos por tipo e filtráveis por dia.
- Rota do dia ligando os pontos na ordem do roteiro.

**Lugares salvos ("quero ir")**
- Lista de lugares salvos da viagem, ainda sem dia definido.
- Importação dos salvos do Google Maps (ver seção 3).
- Agrupamento por proximidade (ver seção 4) e arrastar um grupo inteiro para um dia do roteiro.

**Gastos**
- Registrar gasto com valor, moeda, categoria, quem pagou e divisão (50/50 ou personalizada).
- Conversão para a moeda base (BRL) com cotação do dia.
- Resumo por categoria, por dia e "quem deve quanto para quem".

**Documentos da viagem**
- Pasta geral da viagem (seguro, vistos, passaportes), além dos anexos de cada item.
- Documentos de reserva disponíveis offline no celular.

**Básico**
- Login só para os dois e-mails de vocês (link mágico por e-mail, sem senha).
- Checklist de bagagem / pré-viagem por viagem.
- Edição compartilhada: o que um adiciona, o outro vê na hora.

## 2. Para depois (V2)

- Encaminhar e-mail de confirmação (ou subir o PDF) e o app criar a reserva sozinho, lendo os dados com IA, como o TripIt faz.
- Alertas de voo (atraso, portão) via API de voos.
- Exportar roteiro para o Google Calendar.
- Modo offline completo e compartilhar roteiro somente leitura com família.
- Estatísticas: países visitados, mapa de todas as viagens.

## 3. Importar os salvos do Google Maps

O Google não oferece API para ler as listas salvas de uma conta, então os caminhos realistas são:

1. **Google Takeout (recomendado)**: em takeout.google.com, exportar "Salvos" e "Maps (seus lugares)".
   - "Lugares com estrela" vem em GeoJSON, já com coordenadas: importação direta.
   - Cada lista ("Quero ir", listas personalizadas) vem como um CSV com nome, nota e link do lugar, mas **sem coordenadas**. O app resolve cada link para coordenadas e endereço usando a API do Google Places (cabe na cota gratuita mensal para o volume de vocês).
   - Vocês sobem o .zip ou os arquivos no app, escolhem quais listas importar e para qual viagem.
2. **Colar link de um lugar**: colar um link do Google Maps cria o lugar no app com nome, endereço e pin.
3. **Busca dentro do app**: buscar "restaurante X Lisboa" e salvar direto.

Link de lista compartilhada do Google Maps não tem formato oficial para leitura; daria para tentar, mas quebra fácil, então fica fora da V1.

## 4. Agrupamento por proximidade

- Algoritmo de agrupamento por densidade (DBSCAN com distância real em km): lugares a até ~1 km um do outro (ajustável por um controle deslizante, ex.: 500 m a 3 km) formam um grupo; lugares isolados ficam em "Avulsos".
- Cada grupo recebe um nome automático pelo bairro (ex.: "Alfama · 6 lugares") e uma cor no mapa.
- Uso prático: "este grupo vira o dia 3", movendo todos de uma vez para o roteiro, já ordenados pela menor caminhada.
- No mapa, zoom afastado junta pins próximos em bolinhas com contagem, para não poluir.

## 5. Stack (atualizada: VPS Hetzner, Go + React)

| Parte | Escolha | Por quê |
|---|---|---|
| Frontend | React + TypeScript com Vite, Tailwind, TanStack Query, como PWA | Instala no celular como app; build estático servido pelo próprio servidor |
| Mapa | MapLibre (react-map-gl) com mapas OpenStreetMap | Gratuito, sem cartão |
| Backend | Go (net/http + chi), API REST em JSON | Binário único, leve, cabe folgado numa VPS pequena |
| Banco | PostgreSQL + PostGIS | Consultas por distância e agrupamento por proximidade direto no banco |
| Migrações e queries | goose + sqlc | SQL tipado, sem ORM |
| Arquivos (documentos) | Disco da VPS, servidos só para usuários logados | Simples; entra no backup junto com o banco |
| Login | E-mail e senha (bcrypt) com cookie de sessão; só as duas contas, criadas por comando | Não depende de serviço de e-mail |
| Sincronização | Atualização em tempo real via SSE | O que um adiciona aparece para o outro na hora |
| Busca/importação de lugares | Google Places API (opcional) ou OpenStreetMap | Places é melhor para restaurantes e lojas; cota grátis cobre o uso de vocês |
| Câmbio | API gratuita de cotações, com cache diário | Conversão dos gastos |
| Deploy | Docker Compose: app Go + Postgres + Caddy (HTTPS automático) | Um `docker compose up -d` na VPS |
| Backup | pg_dump + arquivos, diário, para Hetzner Storage Box ou outro destino | Não perder documentos de viagem |
| CI | GitHub Actions: testes, build da imagem e deploy por SSH na VPS | Cada merge vai para o ar sozinho |

**Custo:** só a VPS que você já tem. O app inteiro deve rodar com folga em 1 vCPU e 2 GB de RAM.

**O que vou precisar de você na hora do deploy:** um domínio ou subdomínio apontando para a VPS (para o HTTPS), acesso SSH para o GitHub Actions (uma chave de deploy) e, se quiser, a chave do Google Places.

## 6. Como seria a entrega

1. Repositório privado no seu GitHub, monorepo com `backend/` (Go) e `frontend/` (React).
2. Etapa 1: login, viagens, roteiro dia a dia, itens com anexos, mapa e deploy na VPS.
3. Etapa 2: lugares salvos, importação do Takeout e agrupamento por proximidade.
4. Etapa 3: gastos e divisão, checklist, modo offline dos documentos.

Cada etapa vem como um pull request para você revisar.
