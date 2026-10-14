# Etapa 5: atualização em tempo real e pins agrupados no mapa (design)

Status: proposta, aguardando o Gustavo
Plano geral: `docs/plano-app-viagens.md`. As etapas 1 a 4 já estão na `main`.

## Por que esta etapa

As quatro etapas do plano estão prontas. Duas coisas do plano original ficaram de fora de todas elas:

- **Sincronização** (seção 5): "o que um adiciona aparece para o outro na hora". Hoje a outra pessoa só vê a mudança quando recarrega a tela ou volta para o app.
- **Bolinhas no mapa** (seção 4): "zoom afastado junta pins próximos em bolinhas com contagem, para não poluir". Hoje, com dezenas de salvos, os pins se sobrepõem.

## Tempo real

- O backend mantém uma lista de quem está com o app aberto (`GET /api/events`, Server-Sent Events, sem biblioteca).
- Toda alteração que dá certo (criar, editar ou apagar item, anexo, gasto, checklist ou viagem) avisa todo mundo conectado. O aviso diz só "mudou", sem os dados.
- O app, ao receber o aviso, busca de novo o que está na tela. Para duas pessoas isso custa quase nada.
- Se a conexão cair (túnel, avião), o navegador reconecta sozinho e o app busca tudo de novo ao voltar.
- Um formulário aberto não perde o que você está digitando: ele só é preenchido na primeira vez, como já funciona hoje.
- O service worker não guarda essa conexão.
- Limite: um processo só (é o que roda na VPS). Se um dia houver mais de um, o aviso passa a precisar do banco.

## Bolinhas no mapa

- Na aba Mapa, nos Salvos e no mapa do item: pins a menos de ~44 px um do outro na tela viram uma bolinha com o número de lugares.
- Tocar na bolinha aproxima o mapa até separar os pins. Os grupos são recalculados a cada movimento ou zoom.
- O mapinha de cada dia continua sem agrupar, porque precisa mostrar a ordem 1, 2, 3.
- Feito à mão no front (agrupamento por grade na tela), sem biblioteca. Os pins atuais, com emoji, número e cor, continuam iguais.

## Testes

- Go: duas conexões em `/api/events` recebem o aviso depois de um `POST` de item; `GET` não avisa; erro 400 não avisa; sem sessão → 401.
- Vitest: `groupPins` junta pontos próximos na tela, separa os distantes, e um pin sozinho continua pin.
- Navegador: duas abas logadas, criar um item em uma e ver aparecer na outra sem recarregar; mapa com muitos salvos mostra bolinhas e o toque aproxima.
