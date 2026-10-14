# Ajustes no celular — plano

**Spec:** `docs/superpowers/specs/2026-10-14-ajustes-celular-design.md`. Execução: nativa.

### Task 1: busca de endereço pelo Photon
**Files:** `backend/geocode.go`, `backend/geocode_test.go`, `backend/auth.go` (`photonURL`).
- [ ] Testes com Photon falso: resultado `{label, lat, lng}` com rótulo montado sem repetição; `lat`/`lng` da consulta repassados como `lat`/`lon`; 500 e lentidão → 502.
- [ ] Implementar; commit.

### Task 2: telas
**Files:** `frontend/src/components/AddressSearch.tsx` (novo), `components/FormBar.tsx` (novo), `pages/ItemSheet.tsx`, `ExpenseForm.tsx`, `TripForm.tsx`, `SavedTab.tsx`, `Trip.tsx`, `index.css`.
- [ ] Busca ao digitar com pausa de 350 ms e cancelamento da busca anterior.
- [ ] Barra "‹ Voltar" no topo e "Cancelar" + "Salvar" embaixo.
- [ ] Grupos de botões iguais; CSS dos campos de data no Safari; "+ Item" acima da barra do iPhone.
- [ ] `npm test && npm run build`; navegador em 375×667; commit.
