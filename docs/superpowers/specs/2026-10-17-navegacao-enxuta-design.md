# Navegação enxuta da viagem

## Problema

A barra inferior da viagem tem seis abas só com texto (Roteiro, Salvos, Mapa, Gastos, Chat, Mais).
No celular elas ficam apertadas, e nada diz o que é principal.

## Decisão

- Barra inferior com quatro destinos, cada um com ícone e rótulo: **Roteiro, Mapa, Gastos, Mais**.
- **Salvos** vira a segunda vista do Roteiro: um seletor "Dias | Salvos" no topo do roteiro.
  São as ideias ainda sem dia, então moram junto do roteiro.
- **Assistente** (o antigo Chat) sai da barra e vira um botão no cabeçalho, ao lado de "‹ Viagens".
  Continua a um toque de qualquer aba.
- **Mais** continua igual por dentro (checklist, documentos, compartilhar, calendário).
- Os endereços antigos continuam valendo: `?aba=salvos`, `?aba=chat` e `?aba=documentos`
  abrem a mesma coisa de antes; só muda qual item da barra fica aceso.

## Visual

Mantém a identidade de painel de embarque: azul `ink`, amarelo `signal`, Barlow Condensed.
O item ativo da barra fica em `ink` com destaque amarelo; os outros em `muted` (ver a revisão abaixo).
Ícones são SVG em linha (traço de 2px), sem biblioteca nova.

## Fora do escopo

Reorganizar o conteúdo de Mais, mudar páginas de item ou gasto.

## Revisão: cantos arredondados (pedido do Gustavo, a partir do gastos.app)

- A barra vira uma pílula flutuante (fundo branco translúcido, sombra suave) e o "+ Item"
  vira um botão redondo ao lado dela, só no Roteiro. O item ativo ganha uma pílula amarela atrás do ícone.
- Cantos maiores no app todo, pelo tema: cartões e linhas 18px, botões e campos 12px.
- O seletor "Dias | Salvos" segue o segmented do gastos.app: trilho cinza, opção ativa branca.
- Cores e fontes do viagens continuam (azul `ink`, amarelo `signal`, Barlow).

## Revisão: menos coisa na tela (também a partir do gastos.app)

- Cabeçalho só com botões redondos de ícone: voltar à esquerda; Assistente e Editar à direita.
- "Baixar para usar offline" sai do topo e vai para Mais › Usar sem internet. O topo só mostra aviso
  quando está sem internet ou com alterações esperando.
- Listas viram blocos brancos com linhas divididas (um bloco por dia), sem a borda colorida de cada item.
- O + redondo cria o que faz sentido na tela: item no Dias, lugar no Salvos, gasto no Gastos.
  Saem os botões "+ Lugar" e "+ Gasto".
- Gastos: total grande sobre o fundo, sem a caixa azul.
- Mais vira um índice de linhas (Checklist, Documentos, Compartilhar, Calendário, Usar sem internet);
  cada uma abre a sua seção (`?aba=mais&secao=`). `?aba=documentos` abre direto em Documentos.
