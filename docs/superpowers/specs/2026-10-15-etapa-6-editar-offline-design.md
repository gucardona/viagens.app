# Etapa 6: editar sem internet (design)

Status: proposta, aguardando o Gustavo (recomendada no cartão de 15/10)
Base: etapa 4 (viagem disponível offline, só leitura).

## O problema

Hoje, sem internet, dá para ver a viagem guardada mas não salvar nada: aparece "Sem internet. Tente de novo quando estiver online." Em viagem, a internet falha justo no metrô, no avião ou no interior, que é quando vocês anotam um gasto, riscam o checklist ou mudam o horário de algo.

## O que muda

- Sem internet, estas ações passam a funcionar e ficam numa fila no celular:
  - item: criar, editar, apagar;
  - gasto: criar, editar, apagar;
  - checklist: adicionar, marcar, apagar.
- A tela mostra a mudança na hora, como se tivesse salvado. A faixa de cima diz "2 alterações esperando internet".
- Quando a internet volta (ou ao abrir o app com internet), a fila é enviada sozinha, na ordem em que foi feita. Depois, a tela busca tudo de novo e a outra pessoa recebe pelo tempo real.
- Um item criado offline pode ser editado ou ganhar gasto ainda offline. O app troca o número provisório pelo definitivo ao enviar.
- Se o servidor recusar uma alteração da fila (ex.: o item foi apagado pela outra pessoa), ela é descartada e a faixa avisa: "1 alteração não pôde ser salva: Item não encontrado".
- Continuam precisando de internet: anexar documento, importar do Google, colar link, buscar endereço, restaurantes e banheiros por perto, copiar checklist, apagar ou editar a viagem.

## Como

- `outbox.ts` no front, sem biblioteca: fila em `localStorage`.
- `api()` passa a avisar quando o erro é falta de rede (não quando o servidor respondeu erro). Só então a ação entra na fila.
- As leituras (`/api/trips/{id}`, gastos, checklist) aplicam a fila por cima do que veio da rede ou da cópia guardada. Assim a mudança aparece mesmo depois de fechar e abrir o app offline.
- Os totais e o saldo dos gastos só se atualizam quando a fila é enviada. Um gasto feito offline aparece na lista com "esperando internet".
- Conflito: vale a última alteração enviada, como já acontece hoje entre vocês dois.

## Testes

- Vitest:
  - aplicar a fila sobre a viagem, os gastos e o checklist (criar, editar, apagar);
  - troca do número provisório pelo definitivo, incluindo o `item_id` de um gasto;
  - envio da fila com `fetch` falso: para na falta de rede, descarta e registra a recusa do servidor.
- Navegador: offline, criar item, gasto e marcar o checklist; recarregar ainda offline e ver tudo; voltar a rede e conferir no servidor.

## Limites

- A fila fica só no celular de quem fez. Se o celular for limpo antes de voltar a internet, as alterações se perdem.
- O iPhone pode apagar dados de apps web pouco usados por semanas (mesmo limite da etapa 4).
