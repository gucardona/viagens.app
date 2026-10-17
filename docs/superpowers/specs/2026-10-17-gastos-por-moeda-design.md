# Gastos: total por moeda

Pedido do Gustavo (com um print de outro app): definir o valor e a moeda de cada gasto e ver um resumo da viagem,
como "Total R$ 18.018,30 + CZK 6.960,00 + € 561,25 + HUF 47.600,00".

## O que já existia

Cada gasto já guarda valor e moeda, e a aba Gastos mostra o total convertido para reais (cotação do dia, Frankfurter),
por categoria e por dia, e quem deve quanto.

## O que muda

- **Resumo por moeda**: logo abaixo do total em reais, uma linha com o total em cada moeda em que se gastou,
  na moeda original, somando como no print ("R$ 1.200,00 + € 561,25 + HUF 47.600,00"). Só aparece quando a viagem
  tem gasto em mais de uma moeda, ou numa moeda que não é o real. Conta também os gastos sem cotação, que hoje ficam fora do total.
- **Qualquer moeda**: o campo Moeda deixa de ser uma lista fechada. Continua sugerindo as moedas comuns
  (agora com CZK, HUF, PLN, DKK, SEK, NOK, TRY, CNY, UYU, COP, PEN) e aceita qualquer código de três letras.
  O backend já aceitava qualquer código ISO; moeda sem cotação no Frankfurter cai no aviso "sem cotação" que já existe.

## Como

Só no front: `byCurrency(expenses)` em `money.ts` soma `amount_cents` por moeda (real primeiro, depois as outras em ordem alfabética, como no print).
O campo Moeda vira `<input list>` com `<datalist>`, convertido para maiúsculas.

Fora: mudar a moeda base (continua o real), resumo por pessoa.
