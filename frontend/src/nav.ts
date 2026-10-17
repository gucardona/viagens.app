export type View = 'roteiro' | 'salvos' | 'mapa' | 'gastos' | 'chat' | 'mais'
export type Bar = 'roteiro' | 'mapa' | 'gastos' | 'chat' | 'mais'

const VIEWS: View[] = ['roteiro', 'salvos', 'mapa', 'gastos', 'chat', 'mais']

// Which screen ?aba= opens and which bottom-bar item it lights. Salvos lives under Roteiro
// and ?aba=documentos predates Mais.
export function navFor(aba: string | null): { view: View; bar: Bar } {
  const view: View = aba === 'documentos' ? 'mais' : VIEWS.find((v) => v === aba) ?? 'roteiro'
  const bar = view === 'salvos' ? 'roteiro' : view
  return { view, bar }
}

// What the round + next to the bar creates on each view; null hides it.
export function addFor(view: View, tripId: string | number): { to: string; label: string } | null {
  if (view === 'roteiro') return { to: `/viagens/${tripId}/itens/novo`, label: 'Novo item' }
  if (view === 'salvos') return { to: `/viagens/${tripId}/itens/novo?salvo=1`, label: 'Novo lugar' }
  if (view === 'gastos') return { to: `/viagens/${tripId}/gastos/novo`, label: 'Novo gasto' }
  return null
}
