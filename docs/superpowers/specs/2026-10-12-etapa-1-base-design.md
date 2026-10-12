# Etapa 1: base do app de viagens (design)

Status: aguardando revisão do Gustavo
Plano geral do produto: `docs/plano-app-viagens.md` (escopo V1 em 3 etapas). Este spec cobre só a **etapa 1**. As etapas 2 (lugares salvos, importação do Google, agrupamento por proximidade) e 3 (gastos, checklist, offline) terão specs próprios.

## Objetivo

Um app web privado para Gustavo e a parceira montarem o roteiro de uma viagem dia a dia, com cada item no mapa e com os documentos da reserva anexados, rodando na VPS Hetzner do Gustavo.

**Pronto quando:** os dois conseguem entrar pelo celular, criar uma viagem, adicionar um hotel e um voo com endereço, código de reserva e PDF, ver os dois no roteiro do dia certo e no mapa, e abrir o PDF com um toque.

## O que o Gustavo pediu vs. o que estou assumindo

Pedido: roteiro com pins no mapa, documentos e endereço nos itens, reservas, só os dois usam, VPS Hetzner, backend em Go, front em React, fácil e intuitivo.

Assumido (corrija se estiver errado):
- Todas as viagens são compartilhadas entre as duas contas. Não há permissão por viagem.
- Interface em português, pensada primeiro para celular.
- Horários são sempre o horário local do lugar (como no voucher). O app não converte fuso.

## Fora da etapa 1 (de propósito)

- Importação do Google Maps, lugares salvos e agrupamento por proximidade (etapa 2).
- Busca de lugares pelo Google Places (etapa 2, junto da importação). Na etapa 1 a busca de endereço usa OpenStreetMap.
- Gastos, divisão, checklist, modo offline (etapa 3).
- Atualização em tempo real. Na etapa 1, a tela recarrega os dados ao voltar para o app (refetch on focus). Para duas pessoas isso basta; se incomodar, entra SSE depois.
- Cadastro pela tela. As duas contas são criadas por um comando na VPS.

## Mudança em relação ao plano: SQLite em vez de PostgreSQL

Para duas pessoas numa VPS só, SQLite resolve com folga e simplifica tudo: sem container de banco, backup é copiar um arquivo, testes rodam sem serviço externo. O agrupamento por proximidade da etapa 2 roda em Go sobre algumas centenas de pontos, sem precisar de PostGIS. Se um dia precisar de Postgres, a troca fica restrita à camada de banco.

## Arquitetura

Um único binário Go serve a API em `/api/*` e o front React já compilado (embutido no binário com `go:embed`). Caddy na frente faz HTTPS.

```
celular/navegador ──HTTPS──> Caddy ──> app Go ──> SQLite (/data/viagens.db)
                                         └──────> arquivos (/data/files/)
                                         └──────> Nominatim (busca de endereço)
```

Repositório (monorepo):
```
backend/    Go: main.go, handlers, banco, migrações SQL embutidas, testes
frontend/   React + TypeScript (Vite)
deploy/     docker-compose.yml, Caddyfile, guia de deploy e backup
docs/       specs e planos
```

### Backend (Go)

- Só biblioteca padrão onde possível: `net/http` com o roteador do Go 1.22+ (`GET /api/trips/{id}`), `database/sql`, `encoding/json`, `log/slog`.
- Dependências: `modernc.org/sqlite` (SQLite em Go puro, sem CGO) e `golang.org/x/crypto/bcrypt`.
- Migrações: arquivos `.sql` numerados embutidos, aplicados em ordem na inicialização e registrados numa tabela `schema_migrations`.
- Subcomandos do binário: `serve` (padrão), `adduser <email>` (pede a senha), `backup <arquivo>` (usa `VACUUM INTO`, seguro com o app rodando).
- Configuração por variáveis de ambiente: `DATA_DIR` (padrão `/data`), `ADDR` (padrão `:8080`), `COOKIE_SECURE` (padrão `true`).

### Login e segurança

- E-mail e senha com bcrypt. Sessão por cookie `HttpOnly`, `Secure`, `SameSite=Lax`, válida por 30 dias. No banco fica só o hash SHA-256 do token.
- Escritas (`POST/PUT/DELETE`) exigem o cabeçalho `X-Requested-With: viagens` além do cookie, o que bloqueia formulários de outros sites (CSRF).
- Limite de tentativas de login: 5 falhas por e-mail em 15 minutos, contadas em memória (um processo só).
- Toda rota `/api/*` exceto login exige sessão. Anexos só são servidos para quem está logado.
- Upload limitado a 25 MB por arquivo. O nome original é guardado no banco; no disco o arquivo fica com o id gerado, nunca com o nome enviado.

### Dados

```
users        id, email (único), password_hash, name, created_at
sessions     token_hash (pk), user_id, expires_at
trips        id, name, destination, start_date, end_date, notes, created_at, updated_at
items        id, trip_id, kind, title,
             start_date, start_time, end_date, end_time,   -- texto local: "2026-11-03", "14:30"; vazios = sem data/sem hora
             address, lat, lng,                            -- lat/lng nulos = sem pin
             confirmation_code, notes, created_at, updated_at
attachments  id, trip_id, item_id (nulo = documento geral da viagem),
             filename, content_type, size, created_at
```

- `kind`: `flight`, `lodging`, `car`, `transport`, `restaurant`, `activity`, `event`, `note`.
- Itens sem `start_date` aparecem numa seção "Sem data" no roteiro.
- Ordem dentro do dia: `start_time`, itens sem hora por último, empate por `created_at`. Sem arrastar para reordenar na etapa 1.
- Hospedagem aparece no dia do check-in e no do check-out.
- Apagar uma viagem apaga itens, anexos e os arquivos no disco. Apagar um item apaga seus anexos.
- Ids: inteiros do SQLite.

### API

```
POST   /api/login                 {email, password} -> cookie
POST   /api/logout
GET    /api/me

GET    /api/trips                 lista (ordenada por start_date)
POST   /api/trips
GET    /api/trips/{id}            viagem + itens + anexos
PUT    /api/trips/{id}
DELETE /api/trips/{id}

POST   /api/trips/{id}/items
PUT    /api/items/{id}
DELETE /api/items/{id}

POST   /api/trips/{id}/attachments    multipart: file, item_id opcional
GET    /api/attachments/{id}          baixa/abre o arquivo
DELETE /api/attachments/{id}

GET    /api/geocode?q=...         repassa para o Nominatim, devolve [{label, lat, lng}]
```

- Erros: status HTTP correto e corpo `{"error": "mensagem em português"}`. Validação na entrada: campos obrigatórios, datas no formato certo, `kind` válido, lat/lng dentro dos limites.
- Geocode: o backend chama o Nominatim com `User-Agent` identificado e no máximo 1 requisição por segundo (regra de uso deles). Se falhar, o front mostra o erro e o usuário ainda pode tocar no mapa para marcar o ponto.

### Front (React)

- Vite + React + TypeScript, `react-router`, TanStack Query (cache e recarregar ao voltar para o app), Tailwind.
- Mapa: `maplibre-gl` direto, com o estilo gratuito do OpenFreeMap (sem chave).
- Instalável na tela inicial (manifest e ícones), sem service worker ainda.

Telas:
1. **Entrar**: e-mail e senha.
2. **Viagens**: cartões com nome, destino e datas; botão "Nova viagem".
3. **Viagem**, com três abas:
   - **Roteiro**: um bloco por dia entre início e fim da viagem, mais "Sem data". Cada item mostra ícone do tipo, horário, título e endereço; tocar abre o item.
   - **Mapa**: pins coloridos por tipo, filtro por dia, linha ligando os itens do dia na ordem do roteiro; tocar no pin mostra o item.
   - **Documentos**: anexos gerais da viagem e de todos os itens; tocar abre o arquivo.
4. **Item** (ver/editar): tipo, título, datas e horas, endereço com busca, mini-mapa onde um toque move o pin, código de reserva, notas, anexos (adicionar foto/PDF pela câmera ou arquivos), botões "Abrir no Google Maps" e "Abrir no Waze".

## Deploy

- `Dockerfile` em etapas: compila o front com Node, compila o Go com o front embutido, imagem final mínima só com o binário.
- `docker-compose.yml`: serviço `app` (volume `/data`) e serviço `caddy` (portas 80/443, HTTPS automático para o domínio em `DOMAIN`).
- GitHub Actions: em todo PR roda testes Go e build do front; em push na `main`, entra na VPS por SSH e roda `git pull && docker compose up -d --build`.
- Backup: guia com um cron diário na VPS que roda `backup` e copia o `.db` e a pasta `files/` para fora da VPS (Hetzner Storage Box ou outro destino que o Gustavo escolher).

## Testes

- Go: testes de API com `httptest` e um SQLite temporário, cobrindo login (certo, errado, limite de tentativas, rota sem sessão), CRUD de viagens e itens com validação, upload e download de anexo (incluindo bloqueio sem login) e apagar viagem removendo os arquivos.
- Front: `tsc` e build no CI, e um teste com Vitest da função que monta os dias do roteiro (ordem, "Sem data", hospedagem em dois dias), que é a lógica mais fácil de errar.
- Verificação manual antes do PR: subir com `docker compose` local, fazer o fluxo do "Pronto quando" no navegador em tamanho de celular.

## Riscos e limites conhecidos

- Nominatim é pior que o Google para achar restaurantes e lojas; o toque no mapa cobre os casos que ele não acha até a etapa 2.
- Sem tempo real: se os dois editarem o mesmo item ao mesmo tempo, vale a última gravação.
- Um processo só (limite de login em memória, SQLite). Para dois usuários isso é o certo; não escala para muitos, e não precisa.
