import { useLocation, useNavigate } from 'react-router-dom'

// useBack goes to the previous screen, or to `fallback` when the page was opened directly.
export function useBack(fallback: string) {
  const nav = useNavigate()
  const { key } = useLocation()
  return () => (key === 'default' ? nav(fallback, { replace: true }) : nav(-1))
}

export function BackBar({ label, onBack }: { label: string; onBack: () => void }) {
  return (
    <div className="sticky top-0 z-20 -mx-4 mb-4 bg-sea/95 px-4 pb-2 pt-[max(0.75rem,env(safe-area-inset-top))] backdrop-blur">
      <button type="button" onClick={onBack} className="-ml-1 flex min-h-11 items-center gap-1 pr-3 font-semibold text-ink">
        <span aria-hidden className="text-2xl leading-none">‹</span> {label}
      </button>
    </div>
  )
}

export function SaveCancel({ save, onCancel }: { save: string; onCancel: () => void }) {
  return (
    <div className="grid grid-cols-2 gap-3">
      <button type="button" onClick={onCancel} className="btn btn-quiet">Cancelar</button>
      <button className="btn">{save}</button>
    </div>
  )
}
