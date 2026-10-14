# Ajustes no celular: botões, voltar/cancelar e busca de endereço (design)

Status: pedido do Gustavo em 14/10, correção (sem decisão de produto nova)

## O que o Gustavo pediu

> na versao mobile nao esta correta a formatacao dos botoes e coisas na tela
> o autocomplete do endereco na hora de adicionar algo nao está funcionando muito bem
> nao tem botao de cancelar ou voltar quando estou adicionando algum item

## O que muda

- **Voltar e cancelar**: item, gasto e viagem ganham uma barra no topo com "‹ Voltar" e, embaixo, "Cancelar" ao lado de "Salvar". Cancelar volta para a tela anterior sem salvar.
- **Botões no celular**:
  - Grupos de botões viram blocos de mesma largura e altura, sem quebrar texto no meio. Isso vale para Ingresso, Divisão do gasto e as ações da aba Salvos.
  - Campos de data e hora seguem a largura da coluna no iPhone (o Safari ignora a largura desses campos).
  - O botão "+ Item" respeita a barra do iPhone e não fica em cima das abas.
- **Busca de endereço**:
  - Passa a sugerir enquanto vocês digitam (a partir de 3 letras, depois de uma pausa curta), sem precisar tocar em "Buscar".
  - Usa o Photon (`photon.komoot.io`), um serviço gratuito sobre o OpenStreetMap feito para busca enquanto se digita. O Nominatim pede para não ser usado assim.
  - Os resultados vêm mais perto da viagem: a busca prioriza a região dos itens que já têm pin.
  - O texto da sugestão fica curto: nome, rua, cidade e país.

## Testes

- Go (Photon falso): formato da resposta vira `{label, lat, lng}`, posição da viagem repassada como `lat`/`lon`, falha e lentidão do Photon dão 502.
- Go: o rótulo não repete partes iguais (ex.: cidade com o mesmo nome do lugar).
- Navegador em 375×667: criar item com Cancelar, sugestões aparecendo ao digitar, grupos de botões sem quebra.

## Limites

- Não há iPhone aqui. O ajuste dos campos de data é a correção conhecida para o Safari, mas só dá para confirmar no aparelho.
