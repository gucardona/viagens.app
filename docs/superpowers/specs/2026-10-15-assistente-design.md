# Assistente de viagem (chat com o Claude)

Pedido do Gustavo: uma aba de chat onde cada um conecta a própria chave da API do Claude e conversa com um consultor de viagens que dá dicas perto de onde estamos, sugere atrações, traz documentos de itens, responde endereços e adiciona coisas no roteiro e nos gastos, por texto ou por voz.

## O que o usuário vê

Na viagem, uma aba nova **Chat** (Roteiro, Salvos, Mapa, Gastos, Chat, Mais).

- Sem chave: um campo "Chave da API do Claude" com uma linha dizendo onde criar (console.anthropic.com › API Keys) e que a chave fica guardada no servidor, só para a sua conta.
- Com chave: a conversa, um campo de texto, o botão de microfone e "Enviar". No rodapé, "Nova conversa" e "Trocar chave".
- A resposta vem em texto. Quando o assistente adiciona algo, aparece um botão para o item ou o gasto criado ("Roteiro: Museu do Prado"). Quando ele traz um documento, aparece o botão para abrir o arquivo.
- Voz: o microfone usa o reconhecimento de fala do navegador (pt-BR, sem serviço pago). O texto aparece no campo enquanto a pessoa fala e é enviado quando ela para. Navegador sem suporte não mostra o botão.
- Localização: ao enviar, o app pede a posição atual ao navegador (uma vez; o navegador lembra). Sem permissão, o assistente usa os lugares do roteiro.
- A conversa fica guardada no aparelho, por viagem, até tocar em "Nova conversa".

## Chave

Por usuário, na coluna `users.anthropic_key`. `GET /api/assistant/key` devolve só `{set, hint}` (os 4 últimos caracteres); `PUT /api/assistant/key {key}` grava, e chave vazia apaga. A chave nunca volta para o navegador e uma conta não vê a da outra.

`// shortcut: a chave fica em texto no banco (e no backup); cifrar se o banco sair da VPS.`

## Conversa

`POST /api/trips/{id}/assistant {messages: [{role, text}], lat?, lng?}` → `{reply, links: [{label, to}], docs: [Attachment]}`.

- O histórico vai do aparelho como texto (até 40 mensagens de até 4000 caracteres). Cada pedido começa uma conversa nova na API só com o texto, sem blocos de raciocínio de pedidos anteriores.
  `// shortcut: o que as ferramentas leram em mensagens anteriores não volta; o assistente lê de novo se precisar.`
- O servidor roda o laço de ferramentas (até 10 voltas) com `claude-sonnet-5-5`, esforço `medium`, `fallbacks: "default"` (outro modelo responde se o classificador recusar) e cache do prompt. Chamada direta por HTTP, sem SDK (biblioteca padrão antes de dependência).
- O prompt de sistema traz: data de hoje, a viagem, quem está falando, as duas contas (para "quem pagou") e a posição atual quando houver. Pede respostas curtas em português, em texto simples.
- Erros da API viram mensagens em português: chave inválida (401), limite (429), indisponível (5xx/529), recusa.

## Ferramentas

As ferramentas chamam as rotas que o app já tem, por dentro do servidor, com a sessão de quem pediu; validação e avisos para a outra tela continuam os mesmos.

| Ferramenta | Faz |
|---|---|
| `get_trip` | `GET /api/trips/{id}`: viagem, itens (endereço, horário, código), documentos |
| `get_expenses` | `GET /api/trips/{id}/expenses`: gastos, totais em reais, quem deve a quem |
| `search_nearby {what, lat, lng}` | `GET /api/nearby`: `restaurant`, `toilets` e o novo `attraction` (museus, mirantes, galerias, atrações turísticas em 1,5 km) |
| `search_place {query}` | `GET /api/geocode`: endereço e coordenadas de um lugar |
| `add_item {...}` | `POST /api/trips/{id}/items` com os campos do item |
| `add_expense {...}` | `POST /api/trips/{id}/expenses`; quem pagou é quem está falando se não for dito |
| `show_documents {ids}` | mostra os documentos na conversa, com botão para abrir |
| `read_document {id}` | manda o PDF, imagem ou texto (até 10 MB) para o assistente ler, para responder "qual o código da reserva?" |

Fora: editar ou apagar itens pelo chat, busca na web, resposta falada, histórico no servidor.
