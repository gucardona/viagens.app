# Exportar o roteiro para o calendário

Item da V2 do plano ("Exportar roteiro para o Google Calendar"), sem API: o app gera um arquivo `.ics` (iCalendar, RFC 5545) que o celular ou o Google Agenda importam.

## O que o usuário vê

Na aba "Mais" da viagem, uma seção "Calendário" com o botão "Baixar para o calendário" e uma linha explicando: no iPhone, tocar no arquivo oferece adicionar todos os eventos; no Google Agenda, use Importar.

## Arquivo

`GET /api/trips/{id}/calendar.ics` (com sessão), `text/calendar`, baixado como `<nome da viagem>.ics`.

- Um evento por item com data de início. Itens sem data (salvos) ficam de fora.
- Sem hora de início: evento de dia inteiro, do início até o fim (ou só o dia de início).
- Com hora: evento com horário; fim = data/hora de fim quando houver, senão 1 hora.
- Hora "flutuante" (sem fuso): o calendário mostra no fuso do aparelho, que durante a viagem é o fuso do destino.
  `// shortcut: horário flutuante; antes da viagem aparece no fuso de casa. Mudar se guardarmos fuso por item.`
- Título = título do item. Local = endereço; GEO quando tem pino.
- Descrição: código de confirmação, "Comprar ingresso" / "Ingresso comprado", "Horário aproximado", observações.
- UID estável por item (`item-<id>@viagens.app`), para que importar de novo atualize em vez de duplicar onde o calendário suporta.
- Escapa `\ ; , ` e quebras de linha; dobra linhas em 75 bytes sem cortar caractere UTF-8; CRLF.

Fora: assinatura de calendário por URL (precisaria de link sem login), lembretes.
