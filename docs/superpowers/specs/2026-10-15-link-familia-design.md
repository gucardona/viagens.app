# Link de leitura para a família

Item da V2 do plano ("compartilhar roteiro somente leitura com família").

## O que o usuário vê

- Aba "Mais" da viagem, seção "Link para a família": "Criar link". Depois de criado: o link num campo, "Enviar link" (menu de compartilhar do celular; no computador copia) e "Desativar link" (com confirmação).
- Quem abre `/compartilhado/<token>` vê, sem login, o nome da viagem, destino e bandeiras, e os dias com horários, endereços, status do ingresso e o mapinha de cada dia. Nada é clicável para editar.

## O que fica de fora

Códigos de reserva, observações (da viagem e dos itens), documentos, gastos, checklist e salvos sem data. Observações costumam ter dados pessoais (passaporte, telefone); se a família precisar, dá para rever.

## Backend

- Migração `005_share.sql`: `trips.share_token` (nulo = sem link), índice único.
- `POST /api/trips/{id}/share`: cria o token (24 bytes aleatórios, base64url) ou devolve o existente. `DELETE` apaga.
- `GET /api/shared/{token}` sem sessão: viagem + itens com data, sem os campos acima. 404 "Link inválido ou desativado".
- O token aparece em `trip.share_token` para quem está logado, para a tela mostrar o link.

## Front

- `Day` e `DayMap` aceitam ficar sem link (modo leitura).
- `live.ts` não abre o canal de tempo real em `/compartilhado/` (sem sessão daria 401 e mandaria para o login).
- A página da família não atualiza sozinha; recarregar mostra as mudanças.
