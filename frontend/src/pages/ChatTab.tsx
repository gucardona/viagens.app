import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, json } from '../api'
import type { Attachment } from '../types'

type Msg = { role: 'user' | 'assistant'; text: string; links?: { label: string; to: string }[]; docs?: Attachment[] }
type Reply = { reply: string; links: Msg['links']; docs: Attachment[] }

const SUGGESTIONS = ['O que tem para ver perto de mim?', 'Qual o endereço do hotel?', 'Me mostra a reserva do hotel', 'Gastei 30 euros no almoço']

// The browser's own speech recognition (Chrome, Safari); no paid service. Missing on Firefox.
type Recognition = { lang: string; interimResults: boolean; start(): void; stop(): void; onresult: (e: { results: ArrayLike<ArrayLike<{ transcript: string }>> }) => void; onend: () => void; onerror: () => void }
const Speech = (window as unknown as { SpeechRecognition?: new () => Recognition; webkitSpeechRecognition?: new () => Recognition }).SpeechRecognition
  ?? (window as unknown as { webkitSpeechRecognition?: new () => Recognition }).webkitSpeechRecognition

// The position now, or null when the browser says no or takes too long. The browser's own timeout
// only starts after the permission prompt is answered, so an ignored prompt needs ours.
const here = () => new Promise<GeolocationCoordinates | null>((resolve) => {
  if (!navigator.geolocation) return resolve(null)
  setTimeout(() => resolve(null), 8000)
  navigator.geolocation.getCurrentPosition((p) => resolve(p.coords), () => resolve(null), { timeout: 5000, maximumAge: 300000 })
})

function load(key: string): Msg[] {
  try { return JSON.parse(localStorage.getItem(key) ?? '[]') } catch { return [] }
}

function KeyForm({ hint, onDone }: { hint?: string; onDone: () => void }) {
  const qc = useQueryClient()
  const [key, setKey] = useState('')
  const [error, setError] = useState('')
  async function save(value: string) {
    setError('')
    try {
      await api('/api/assistant/key', json('PUT', { key: value }))
      await qc.invalidateQueries({ queryKey: ['assistant-key'] })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }
  return (
    <form className="mt-4 space-y-3 rounded-lg bg-paper p-4" onSubmit={(e) => { e.preventDefault(); if (key.trim()) save(key) }}>
      <label className="block font-semibold text-ink" htmlFor="claude-key">Chave da API do Claude</label>
      <p className="text-sm text-muted">
        Crie em console.anthropic.com, em API Keys. Ela fica guardada no servidor, só para a sua conta, e o uso é cobrado na sua conta da Anthropic.
      </p>
      <input id="claude-key" className="field" type="password" autoComplete="off" placeholder={hint ? `Atual termina em ${hint}` : 'sk-ant-…'} value={key} onChange={(e) => setKey(e.target.value)} />
      {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
      <div className="flex flex-wrap gap-3">
        <button className="btn" disabled={!key.trim()}>Salvar chave</button>
        {hint && <button type="button" className="btn btn-quiet" onClick={onDone}>Cancelar</button>}
        {hint && <button type="button" className="btn-danger px-2 font-semibold" onClick={() => confirm('Remover sua chave?') && save('')}>Remover chave</button>}
      </div>
    </form>
  )
}

export default function ChatTab({ tripId }: { tripId: number }) {
  const store = `chat-${tripId}`
  const { data: key } = useQuery({ queryKey: ['assistant-key'], queryFn: () => api<{ set: boolean; hint: string }>('/api/assistant/key') })
  const [editKey, setEditKey] = useState(false)
  const [msgs, setMsgs] = useState<Msg[]>(() => load(store))
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [listening, setListening] = useState(false)
  const rec = useRef<Recognition | null>(null)
  const end = useRef<HTMLDivElement>(null)

  useEffect(() => { try { localStorage.setItem(store, JSON.stringify(msgs.slice(-40))) } catch { /* full or blocked: the chat still works */ } }, [store, msgs])
  useEffect(() => { end.current?.scrollIntoView({ block: 'end' }) }, [msgs, busy])

  async function send(text: string) {
    text = text.trim()
    if (!text || busy) return
    const next = [...msgs, { role: 'user' as const, text }]
    setMsgs(next)
    setInput('')
    setError('')
    setBusy(true)
    try {
      const pos = await here()
      const res = await api<Reply>(`/api/trips/${tripId}/assistant`, json('POST', {
        messages: next.slice(-40).map(({ role, text }) => ({ role, text })),
        ...(pos && { lat: pos.latitude, lng: pos.longitude }),
      }))
      setMsgs([...next, { role: 'assistant', text: res.reply || '…', links: res.links, docs: res.docs }])
    } catch (err) {
      setMsgs(msgs)
      setInput(text)
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  function talk() {
    if (listening) return rec.current?.stop()
    if (!Speech) return
    const r = new Speech()
    r.lang = 'pt-BR'
    r.interimResults = true
    let said = ''
    r.onresult = (e) => {
      said = Array.from(e.results, (res) => res[0].transcript).join('')
      setInput(said)
    }
    r.onerror = () => setError('Não consegui ouvir. Confira a permissão do microfone.')
    r.onend = () => { setListening(false); rec.current = null; if (said.trim()) send(said) }
    rec.current = r
    setError('')
    setListening(true)
    r.start()
  }

  if (!key) return null
  if (!key.set || editKey) return <KeyForm hint={key.set ? key.hint : undefined} onDone={() => setEditKey(false)} />

  return (
    <div className="pb-24">
      {msgs.length === 0 && (
        <div className="mt-4">
          <p className="text-muted">Peça dicas perto de você, endereços, documentos, ou para pôr algo no roteiro e nos gastos. Pode falar no microfone.</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {SUGGESTIONS.map((s) => <button key={s} className="rounded-full border border-line bg-paper px-3 py-1.5 text-sm text-ink" onClick={() => send(s)}>{s}</button>)}
          </div>
        </div>
      )}
      <ul className="mt-4 space-y-3" aria-live="polite">
        {msgs.map((m, i) => (
          <li key={i} className={m.role === 'user' ? 'flex justify-end' : ''}>
            <div className={`max-w-[85%] rounded-2xl px-3 py-2 ${m.role === 'user' ? 'bg-ink text-white' : 'bg-paper'}`}>
              <p className="whitespace-pre-wrap">{m.text}</p>
              {(m.links?.length || m.docs?.length) ? (
                <div className="mt-2 flex flex-wrap gap-2">
                  {m.links?.map((l) => <Link key={l.to} to={l.to} className="rounded bg-signal/40 px-2 py-1 text-sm font-semibold text-ink">{l.label}</Link>)}
                  {m.docs?.map((d) => <a key={d.id} href={`/api/attachments/${d.id}`} target="_blank" rel="noreferrer" className="rounded bg-sea px-2 py-1 text-sm font-semibold text-ink underline">📄 {d.filename}</a>)}
                </div>
              ) : null}
            </div>
          </li>
        ))}
        {busy && <li className="text-muted">Pensando…</li>}
      </ul>
      {error && <p role="alert" className="mt-3 text-sm text-red-700">{error}</p>}
      <div className="mt-6 flex gap-4 text-sm">
        {msgs.length > 0 && <button className="text-ink underline" onClick={() => { setMsgs([]); setError('') }}>Nova conversa</button>}
        <button className="text-muted underline" onClick={() => setEditKey(true)}>Trocar chave</button>
      </div>
      <div ref={end} />

      <form
        className="fixed inset-x-0 bottom-[calc(5.1rem+env(safe-area-inset-bottom))] z-10 border-t border-line bg-sea/95 py-2 backdrop-blur"
        onSubmit={(e) => { e.preventDefault(); send(input) }}
      >
        <div className="mx-auto flex max-w-2xl gap-2 px-4">
          {Speech && (
            <button type="button" onClick={talk} disabled={busy} aria-label={listening ? 'Parar de ouvir' : 'Falar'}
              className={`btn shrink-0 px-3 ${listening ? 'bg-red-700 animate-pulse' : 'btn-quiet'}`}>🎤</button>
          )}
          <input className="field min-w-0 flex-1" value={input} onChange={(e) => setInput(e.target.value)}
            placeholder={listening ? 'Ouvindo…' : 'Pergunte ou peça algo'} aria-label="Mensagem" enterKeyHint="send" />
          <button className="btn shrink-0" disabled={busy || !input.trim()}>Enviar</button>
        </div>
      </form>
    </div>
  )
}
