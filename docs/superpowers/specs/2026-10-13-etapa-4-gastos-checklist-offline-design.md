# Etapa 4: gastos, checklist e documentos offline (design)

Status: aguardando aprovação do Gustavo
Plano geral: `docs/plano-app-viagens.md` (etapa 4). Etapas 1 a 3 já estão na `main`.

## O que estava no plano

- Gastos com valor, moeda, categoria, quem pagou e divisão; conversão para BRL com a cotação do dia; resumo por categoria, por dia e "quem deve quanto para quem".
- Checklist de bagagem / pré-viagem por viagem.
- Documentos da viagem disponíveis offline no celular.

## Como estou interpretando (corrija se estiver errado)

- **Só duas pessoas** dividem gastos: as duas contas do app. A divisão é "metade cada" (padrão), "só de quem pagou" ou uma porcentagem.
- **Cotação**: Frankfurter (dados do Banco Central Europeu, gratuito, sem chave), pela data do gasto. Ele cobre cerca de 30 moedas (EUR, USD, GBP, JPY, CHF e outras); as que ficam de fora, como o peso argentino, pedem a cotação digitada no gasto.
- **Offline** vale para a viagem inteira que vocês marcarem: roteiro, salvos e todos os documentos. Sem internet, dá para ver tudo; editar continua precisando de internet.

## Gastos

### Dados (migração `004`)

```
expenses  id, trip_id, item_id (nulo), title, amount_cents, currency (ISO, ex. "EUR"),
          category, paid_by (user id), payer_share (0–100, padrão 50),
          date ("2026-11-03"), rate (BRL por 1 unidade; nulo = buscar), created_at
rates     date, currency, brl   -- cache das cotações (pk: date, currency)
```

- Categorias fixas: `lodging` Hospedagem, `transport` Transporte, `food` Comida, `activity` Passeios, `shopping` Compras, `other` Outros.
- `payer_share` é a parte de quem pagou: 50 = metade cada, 100 = só de quem pagou.

### API

```
GET    /api/trips/{id}/expenses     lista + resumo
POST   /api/trips/{id}/expenses
PUT    /api/expenses/{id}
DELETE /api/expenses/{id}
GET    /api/users                   as duas contas (id, nome), para "quem pagou"
```

- O resumo vem calculado pelo backend: total em BRL, por categoria, por dia, e o saldo ("Ana deve R$ 312,40 para Gustavo"). Gasto em BRL usa cotação 1.
- Cotação: o backend busca `api.frankfurter.app/{data}?from={moeda}&to=BRL` na primeira vez que precisa e guarda em `rates`. Data no futuro usa a mais recente. Se o Frankfurter estiver fora ou não cobrir a moeda e o gasto não tiver `rate`, o gasto aparece no valor original e fica fora do total, com o aviso "sem cotação".

### Telas

- Nova aba **Gastos** na viagem: no topo o total em BRL e o saldo entre vocês; depois resumo por categoria (barras horizontais simples) e a lista por dia. Botão "+ Gasto".
- Formulário: valor, moeda (lembra a última usada na viagem), descrição, categoria, data (padrão hoje se estiver dentro da viagem, senão o início), quem pagou, divisão (Metade / Só quem pagou / %), cotação opcional.
- No item do roteiro: "Adicionar gasto" abre o formulário já com o item, a descrição e a categoria (hotel → Hospedagem, voo/trem/carro → Transporte, restaurante → Comida, passeio/evento → Passeios). O item mostra o total gasto nele.
- Abas ficam: Roteiro, Salvos, Mapa, Gastos e "Mais" (Documentos e Checklist), para caber no celular.

## Checklist

- Tabela `checklist (id, trip_id, text, done, position)`.
- Na aba Mais > Checklist: lista com caixa de marcar, campo para adicionar, apagar com um toque longo ou botão, arrastar não (ordem de criação).
- "Copiar de outra viagem": copia os textos (desmarcados) de uma viagem anterior. Sem modelos prontos.
- API: `GET/POST /api/trips/{id}/checklist`, `PUT/DELETE /api/checklist/{id}`, `POST /api/trips/{id}/checklist/copy {from_trip_id}`.

## Documentos offline

- Um service worker escrito à mão (`sw.js`, sem biblioteca):
  - guarda os arquivos do app (HTML, JS, CSS) para abrir sem internet;
  - para chamadas `GET /api/...`, tenta a rede e, sem rede, devolve a última resposta guardada;
  - anexos (`/api/attachments/{id}`) ficam guardados depois de abertos uma vez.
- Na viagem, botão **"Baixar para usar offline"**: busca a viagem, os gastos, o checklist e todos os anexos, e mostra "Disponível offline · 12 documentos · 8 MB" com a data. Abrir a viagem de novo com internet atualiza a cópia.
- Sem internet, aparece uma faixa "Sem internet: mostrando a cópia de 13/11 14:20" e os botões de salvar avisam que precisa de internet.
- A cópia offline fica só no navegador do celular de vocês.

## Testes

- Go: CRUD de gastos com validação (valor > 0, moeda de 3 letras, categoria, `payer_share` 0–100, quem pagou existe); resumo (total, categorias, dias, saldo nos dois sentidos, gasto sem cotação fora do total); cotação buscada uma vez e guardada, Frankfurter falso fora do ar; checklist CRUD e cópia.
- Vitest: formatação de dinheiro em pt-BR, categoria sugerida pelo tipo do item.
- Navegador em 390×844: criar gastos em EUR e BRL, ver saldo; checklist; marcar viagem offline, desligar a rede no navegador e abrir roteiro e um PDF.

## Riscos e limites

- O Frankfurter só tem cotações de dias úteis; fim de semana usa a do último dia útil (é o que a API devolve).
- Moedas fora do Frankfurter (ex.: peso argentino) precisam da cotação digitada.
- Service worker só funciona em HTTPS (o seu domínio já tem) e no iPhone a cópia offline pode ser apagada pelo sistema se o app ficar semanas sem abrir.
- Sem edição offline: se precisar, entra depois com fila de envio.
