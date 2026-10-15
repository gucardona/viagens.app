# Deploy na VPS (Hetzner)

O app (Go + front embutido) roda com Docker Compose e escuta só em `127.0.0.1:8005`. O HTTPS vem do Caddy instalado na VPS, o mesmo dos outros sites. Os dados ficam em `deploy/data/` (banco `viagens.db` e a pasta `files/` com os documentos).

## 1. Antes de começar

- Docker e o plugin `docker compose` instalados na VPS.
- Um domínio ou subdomínio com registro **A** apontando para o IP da VPS (ex.: `viagens.seudominio.com`).
- Caddy instalado na VPS (é ele que cuida do HTTPS) e portas 80 e 443 liberadas no firewall da Hetzner.

## 2. Clonar o repositório

O repositório é público, então o clone é por HTTPS, sem chave. O deploy automático (passo 5) espera o código em `~/gupa.dev/viagens.app`:

```sh
git clone https://github.com/gucardona/viagens.app.git ~/gupa.dev/viagens.app
```

## 3. Subir o app

```sh
cd ~/gupa.dev/viagens.app/deploy
docker compose up -d --build
```

No `/etc/caddy/Caddyfile` da VPS, acrescente:

```caddy
viagens.gupa.dev {
        reverse_proxy 127.0.0.1:8005
}
```

E recarregue com `sudo systemctl reload caddy`. O Caddy emite o certificado na primeira visita.

## 4. Criar as duas contas

```sh
cd ~/gupa.dev/viagens.app/deploy
docker compose exec app /viagens adduser gustavo@email.com "Gustavo"
```

Quando aparecer `Senha:`, digite a senha (pelo menos 8 caracteres) e aperte Enter. Repita com o e-mail e o nome da outra pessoa.

## 5. Deploy automático (GitHub Actions)

A cada push na `main`, depois dos testes, o Actions entra na VPS por SSH e roda `git pull` + `docker compose up -d --build`, e apaga as imagens antigas.

1. Na VPS, crie uma chave para o Actions, autorize-a e gere a linha que identifica a VPS (troque `SEU_IP` pelo IP da VPS):
   ```sh
   ssh-keygen -t ed25519 -f ~/.ssh/viagens_actions -N "" -C viagens-actions
   cat ~/.ssh/viagens_actions.pub >> ~/.ssh/authorized_keys
   cat ~/.ssh/viagens_actions        # chave privada, para DEPLOY_KEY
   ssh-keyscan SEU_IP 2>/dev/null    # linhas para DEPLOY_KNOWN_HOSTS
   ```
2. No GitHub: **Settings > Secrets and variables > Actions > New repository secret**, crie:
   - `DEPLOY_HOST`: o IP da VPS (o mesmo usado no `ssh-keyscan`)
   - `DEPLOY_USER`: o seu usuário na VPS (o que roda `docker` sem sudo)
   - `DEPLOY_KEY`: a chave privada inteira, das linhas `-----BEGIN` até `-----END`
   - `DEPLOY_KNOWN_HOSTS`: a saída do `ssh-keyscan`

Sem `DEPLOY_HOST` o passo de deploy é pulado e só os testes rodam. O Actions só conecta se a VPS apresentar a chave registrada em `DEPLOY_KNOWN_HOSTS`.

## 6. Backup diário

O comando `backup` grava uma cópia consistente do banco mesmo com o app rodando. Os documentos são arquivos comuns em `data/files/`. Exemplo de cron (`crontab -e`) copiando os dois para um Hetzner Storage Box:

```cron
0 3 * * * cd ~/gupa.dev/viagens.app/deploy && docker compose exec -T app /viagens backup /data/backup.db && rsync -a -e "ssh -p 23" data/backup.db data/files uXXXXX@uXXXXX.your-storagebox.de:viagens/
```

Para restaurar: pare o app (`docker compose stop app`), copie `backup.db` para `data/viagens.db`, devolva a pasta `files/` e suba de novo.

## 7. Mapa do Google (opcional)

O mapa usa OpenStreetMap (OpenFreeMap) por padrão. Para trocar pelo Google, o app pede os quadradinhos do mapa à **Map Tiles API** do Google; todo o resto (pins, bolinhas, linha do roteiro, busca por perto) continua igual. A troca é na hora do build, por duas variáveis.

1. Em <https://console.cloud.google.com>, crie um projeto (ex.: `viagens`) e ative o faturamento nele. O Google pede cartão, mas o uso de duas pessoas fica dentro da cota grátis mensal da Map Tiles API.
2. Em **APIs e serviços > Biblioteca**, ative a **Map Tiles API**.
3. Em **APIs e serviços > Credenciais > Criar credenciais > Chave de API**, copie a chave.
4. Na mesma chave, em **Restrições**:
   - **Restrições de aplicativo**: *Sites*, com `https://viagens.gupa.dev/*` (e `http://localhost:5173/*` se for testar local).
   - **Restrições de API**: só a **Map Tiles API**.
   A chave vai no código do site e qualquer um pode vê-la; são essas restrições que impedem o uso fora do app.
5. Opcional: em **Cotas**, limite as requisições por dia da Map Tiles API para nunca passar da cota grátis.
6. Na VPS, crie (ou complete) o arquivo `deploy/.env`, que não vai para o git:
   ```sh
   VITE_MAP_PROVIDER=google
   VITE_GOOGLE_MAPS_KEY=sua-chave
   ```
7. Rode `docker compose up -d --build` (ou espere o próximo deploy automático).

Para voltar ao OpenStreetMap, troque para `VITE_MAP_PROVIDER=osm` (ou apague a linha) e faça o build de novo. Local, as mesmas linhas vão em `frontend/.env.local` antes do `npm run dev`.

## Rodar localmente

```sh
cd backend && mkdir -p tmp
echo "segredo123" | DATA_DIR=./tmp go run . adduser voce@email.com "Você"
COOKIE_SECURE=false DATA_DIR=./tmp go run .        # API em :8080
cd ../frontend && npm install && npm run dev        # front em :5173, com proxy para a API
```
