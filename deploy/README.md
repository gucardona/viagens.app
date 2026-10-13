# Deploy na VPS (Hetzner)

Tudo roda com Docker Compose: o app (Go + front embutido) e o Caddy, que cuida do HTTPS sozinho. Os dados ficam em `deploy/data/` (banco `viagens.db` e a pasta `files/` com os documentos).

## 1. Antes de começar

- Docker e o plugin `docker compose` instalados na VPS.
- Um domínio ou subdomínio com registro **A** apontando para o IP da VPS (ex.: `viagens.seudominio.com`).
- Portas 80 e 443 liberadas no firewall da Hetzner.

## 2. Clonar o repositório privado

Na VPS, crie uma chave só para leitura do repositório:

```sh
ssh-keygen -t ed25519 -f ~/.ssh/viagens_repo -N ""
cat ~/.ssh/viagens_repo.pub
```

No GitHub: **Settings > Deploy keys > Add deploy key**, cole a chave pública, sem permissão de escrita. Depois:

```sh
GIT_SSH_COMMAND="ssh -i ~/.ssh/viagens_repo" git clone git@github.com:gucardona/viagens.app.git ~/viagens.app
cd ~/viagens.app && git config core.sshCommand "ssh -i ~/.ssh/viagens_repo"
```

## 3. Subir o app

```sh
cd ~/viagens.app/deploy
cp .env.example .env    # edite DOMAIN
docker compose up -d --build
```

O Caddy emite o certificado na primeira visita. Abra `https://SEU_DOMINIO`.

## 4. Criar as duas contas

Para a senha não ficar no histórico do shell:

```sh
cd ~/viagens.app/deploy
read -rs SENHA && echo "$SENHA" | docker compose exec -T app /viagens adduser gustavo@email.com "Gustavo"
read -rs SENHA && echo "$SENHA" | docker compose exec -T app /viagens adduser ela@email.com "Nome dela"
```

A senha precisa de pelo menos 8 caracteres.

## 5. Deploy automático (GitHub Actions)

A cada push na `main`, depois dos testes, o Actions entra na VPS por SSH e roda `git pull` + `docker compose up -d --build`.

1. Na VPS, crie uma chave para o Actions e autorize-a:
   ```sh
   ssh-keygen -t ed25519 -f ~/.ssh/viagens_actions -N ""
   cat ~/.ssh/viagens_actions.pub >> ~/.ssh/authorized_keys
   cat ~/.ssh/viagens_actions   # copie a chave privada
   ```
2. No GitHub: **Settings > Secrets and variables > Actions**, crie:
   - `DEPLOY_HOST`: IP ou domínio da VPS
   - `DEPLOY_USER`: usuário SSH (o mesmo que clonou o repositório)
   - `DEPLOY_KEY`: a chave privada copiada acima

Sem `DEPLOY_HOST` o passo de deploy é pulado e só os testes rodam.

## 6. Backup diário

O comando `backup` grava uma cópia consistente do banco mesmo com o app rodando. Os documentos são arquivos comuns em `data/files/`. Exemplo de cron (`crontab -e`) copiando os dois para um Hetzner Storage Box:

```cron
0 3 * * * cd ~/viagens.app/deploy && docker compose exec -T app /viagens backup /data/backup.db && rsync -a -e "ssh -p 23" data/backup.db data/files uXXXXX@uXXXXX.your-storagebox.de:viagens/
```

Para restaurar: pare o app (`docker compose stop app`), copie `backup.db` para `data/viagens.db`, devolva a pasta `files/` e suba de novo.

## Rodar localmente

```sh
cd backend && mkdir -p tmp
echo "segredo123" | DATA_DIR=./tmp go run . adduser voce@email.com "Você"
COOKIE_SECURE=false DATA_DIR=./tmp go run .        # API em :8080
cd ../frontend && npm install && npm run dev        # front em :5173, com proxy para a API
```
