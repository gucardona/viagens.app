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
O item ativo da barra fica aceso com a faixa amarela no topo e o ícone em `ink`; os outros em `muted`.
Ícones são SVG em linha (traço de 2px), sem biblioteca nova.

## Fora do escopo

Reorganizar o conteúdo de Mais, mudar páginas de item ou gasto.
