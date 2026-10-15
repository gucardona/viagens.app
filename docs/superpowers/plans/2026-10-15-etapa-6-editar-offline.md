# Etapa 6: editar sem internet — plano

**Spec:** `docs/superpowers/specs/2026-10-15-etapa-6-editar-offline-design.md`. Execução: nativa.

### Task 1: fila (`outbox.ts`)
- [x] Vitest: `classify`, `queue` (número provisório, edição e exclusão de algo ainda não criado), `overlay` (viagem, gastos, checklist), `flush` (ordem, troca do `item_id`, recusa descartada, parada na falta de rede).

### Task 2: ligar ao app
**Files:** `api.ts` (fila na falta de rede, `overlay` nas leituras), `live.ts` (envia ao abrir, ao voltar a rede e ao reconectar), `main.tsx` (`networkMode: 'always'`), `components/OfflineBar.tsx` (contagem e recusas), marcas "esperando internet" nas listas.
- [x] Navegador: offline, criar item, gasto e marcar checklist; recarregar offline; voltar a rede e conferir no servidor.
