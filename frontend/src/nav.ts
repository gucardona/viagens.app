export type View = 'roteiro' | 'salvos' | 'mapa' | 'gastos' | 'chat' | 'mais'
export type Bar = 'roteiro' | 'mapa' | 'gastos' | 'mais'

const VIEWS: View[] = ['roteiro', 'salvos', 'mapa', 'gastos', 'chat', 'mais']

// Which screen ?aba= opens and which bottom-bar item it lights. Salvos lives under Roteiro,
// the assistant opens from the header (no bar item), and ?aba=documentos predates Mais.
export function navFor(aba: string | null): { view: View; bar: Bar | null } {
  const view: View = aba === 'documentos' ? 'mais' : VIEWS.find((v) => v === aba) ?? 'roteiro'
  const bar = view === 'salvos' ? 'roteiro' : view === 'chat' ? null : view
  return { view, bar }
}
