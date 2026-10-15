# Revisão geral: correções

Revisão do app inteiro (backend e front) atrás de bugs, depois da etapa 6. Só correções, sem funcionalidade nova.

## Front (fila offline)

1. Editar ou apagar algo criado offline, depois que a fila já o enviou, ia para o id temporário e se perdia. Agora a fila guarda o mapa id temporário → id real (`outbox-ids`) e reescreve caminho e `item_id`.
2. Uma edição feita enquanto a fila enviava era sobrescrita ao salvar o resultado. `reconcile` junta o que foi enviado com o que entrou no meio (vira PUT/DELETE no id real).
3. Ids temporários podiam colidir (dois criados no mesmo milissegundo). Agora são sempre negativos e decrescentes.
4. Abrir um gasto ou item com id temporário depois do envio dava "não encontrado". Agora redireciona para o id real; anexos só aparecem para itens já enviados.
5. A barra offline não atualizava os anexos guardados quando chegava um anexo novo.

## Backend

6. Login: logins em paralelo passavam do limite de 5 tentativas (checagem e contagem em locks separados). Agora reserva a tentativa no mesmo lock; e-mail desconhecido também paga o bcrypt (não dá para descobrir contas pelo tempo).
7. Mudar ou tirar o pino de um item mantinha país e bairro antigos. O UPDATE limpa `country_code`, `area` e `geo_key` quando lat/lng mudam.
8. Cotação manual absurda (1e300) estourava o total. Limite de 100000.
9. Moeda sem cotação (ex.: ARS no Frankfurter) era consultada uma vez por gasto a cada listagem. Falhas ficam 10 min em memória; a cotação de hoje fica 1 h em memória e não vai mais para a tabela (antes ficava congelada para sempre).
10. `lat=NaN` passava na validação de "por perto" e ia para o Overpass.
11. Checklist de viagem inexistente devolvia 200 com lista vazia; agora 404.
12. Anexo cujo arquivo sumiu do disco dava 500; agora 404.

Fora: erro de status de migração em `db.go` e corrida de upload durante exclusão da viagem (raros, sem impacto prático para duas pessoas).
