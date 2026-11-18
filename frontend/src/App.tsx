import { useState, type FormEvent } from 'react'
import StatsPanel from './components/StatsPanel'

const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

interface ShortLink {
  code: string
  short_url: string
  long_url: string
}

export default function App() {
  const [url, setUrl] = useState('')
  const [links, setLinks] = useState<ShortLink[]>([])
  const [selected, setSelected] = useState<ShortLink | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const response = await fetch(`${API_URL}/api/shorten`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
      })

      if (!response.ok) {
        const body = (await response.json()) as { error?: string }
        throw new Error(body.error ?? 'No se pudo acortar la URL')
      }

      const created = (await response.json()) as ShortLink
      setLinks((current) => [created, ...current.filter((link) => link.code !== created.code)])
      setUrl('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error inesperado')
    } finally {
      setLoading(false)
    }
  }

  async function handleCopy(text: string) {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(text)
      window.setTimeout(() => setCopied(''), 1500)
    } catch {
      setCopied('')
    }
  }

  return (
    <>
      <style>{styles}</style>
      <div className="app">
        <header className="header">
          <div className="brand">
            <span className="brand-mark">bit</span>
            <span className="brand-sub">¿cómo funciona?</span>
          </div>
          <span className="badge">Ejercicio 0001 / 1024</span>
        </header>

        <main className={selected ? 'layout with-panel' : 'layout'}>
          <section className="content">
            <h1>Acordá un link.</h1>
            <p className="lead">
              Pegás una URL larga, el sistema genera un código corto, redirige desde caché y registra
              cada click para después ver las estadísticas.
            </p>

            <form onSubmit={handleSubmit} className="shorten-form">
              <input
                type="url"
                value={url}
                onChange={(event) => setUrl(event.target.value)}
                placeholder="https://ejemplo.com/un/camino/bastante/largo"
                aria-label="URL a acortar"
                required
              />
              <button type="submit" disabled={loading}>
                {loading ? 'Acortando…' : 'Acortar'}
              </button>
            </form>

            {error && <p className="error">{error}</p>}

            <div className="links">
              <div className="links-head">
                <h2>Links generados</h2>
                <span className="muted">{links.length} en esta sesión</span>
              </div>

              {links.length === 0 && (
                <div className="empty">
                  <span className="empty-icon">↗</span>
                  <p>Todavía no generaste ningún link. Probá con la URL de arriba.</p>
                </div>
              )}

              {links.map((link) => (
                <article className="link-card" key={link.code}>
                  <div className="link-main">
                    <a href={link.short_url} target="_blank" rel="noreferrer">
                      {link.short_url}
                    </a>
                    <p className="original" title={link.long_url}>
                      {link.long_url}
                    </p>
                  </div>
                  <div className="link-actions">
                    <button className="secondary" onClick={() => void handleCopy(link.short_url)}>
                      {copied === link.short_url ? 'Copiado ✓' : 'Copiar'}
                    </button>
                    <button className="secondary" onClick={() => setSelected(link)}>
                      Ver stats →
                    </button>
                  </div>
                </article>
              ))}
            </div>

            <footer className="footer">
              Go · Postgres · Redis · Kafka · ClickHouse — todo levantado con <code>docker compose up</code>.
            </footer>
          </section>

          {selected && (
            <StatsPanel
              apiUrl={API_URL}
              code={selected.code}
              shortUrl={selected.short_url}
              onClose={() => setSelected(null)}
            />
          )}
        </main>
      </div>
    </>
  )
}

const styles = `
  :root {
    color-scheme: dark;
    font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
    background: #0b1020;
    color: #e8ecf7;
  }
  * { box-sizing: border-box; }
  body { margin: 0; }
  button, input { font: inherit; }
  a { color: inherit; }
  .app { min-height: 100vh; padding: 28px clamp(20px, 5vw, 64px) 48px; }
  .header {
    display: flex; align-items: center; justify-content: space-between;
    padding-bottom: 22px; border-bottom: 1px solid #202942;
  }
  .brand { display: flex; align-items: baseline; gap: 10px; }
  .brand-mark {
    font-size: 26px; font-weight: 800; letter-spacing: -1px;
    background: linear-gradient(90deg, #5eead4, #60a5fa);
    -webkit-background-clip: text; background-clip: text; color: transparent;
  }
  .brand-sub { color: #8b96b5; font-size: 14px; }
  .badge {
    font-size: 12px; color: #93c5fd; border: 1px solid #26365c;
    padding: 6px 12px; border-radius: 999px; background: #111a30;
  }
  .layout { display: grid; grid-template-columns: minmax(0, 1fr); gap: 28px; max-width: 1400px; margin: 0 auto; }
  .layout.with-panel { grid-template-columns: minmax(0, 1fr) 380px; }
  .content { padding-top: 48px; max-width: 760px; }
  h1 { font-size: clamp(34px, 5vw, 52px); margin: 0 0 12px; letter-spacing: -1.5px; }
  .lead { color: #a3adc9; line-height: 1.6; max-width: 620px; margin: 0 0 28px; }
  .shorten-form { display: flex; gap: 10px; }
  .shorten-form input {
    flex: 1; min-width: 0; padding: 14px 16px; border-radius: 10px;
    border: 1px solid #2b3758; background: #121a2f; color: #e8ecf7; outline: none;
  }
  .shorten-form input:focus { border-color: #5eead4; box-shadow: 0 0 0 3px #5eead422; }
  .shorten-form button, .link-actions button {
    cursor: pointer; border: none; border-radius: 10px; padding: 14px 20px;
    background: #5eead4; color: #06251f; font-weight: 700; white-space: nowrap;
  }
  .shorten-form button:disabled { opacity: 0.6; cursor: wait; }
  .error { color: #fca5a5; background: #3b1620; border: 1px solid #7f1d2e; padding: 10px 14px; border-radius: 8px; }
  .muted { color: #7f8aa8; font-size: 13px; }
  .links { margin-top: 40px; }
  .links-head { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 12px; }
  .links-head h2 { font-size: 18px; margin: 0; }
  .empty {
    border: 1px dashed #2b3758; border-radius: 12px; padding: 32px;
    text-align: center; color: #8b96b5; background: #0e1528;
  }
  .empty-icon { font-size: 28px; color: #5eead4; }
  .link-card {
    display: flex; align-items: center; justify-content: space-between; gap: 16px;
    border: 1px solid #243050; background: #101830; border-radius: 12px;
    padding: 16px 18px; margin-bottom: 10px;
  }
  .link-main { min-width: 0; }
  .link-main a { color: #5eead4; font-weight: 700; text-decoration: none; }
  .link-main a:hover { text-decoration: underline; }
  .original {
    margin: 6px 0 0; color: #8b96b5; font-size: 13px;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 44ch;
  }
  .link-actions { display: flex; gap: 8px; }
  .link-actions .secondary {
    background: #1b2542; color: #cfd8ee; border: 1px solid #2b3758;
    padding: 10px 12px; font-weight: 600; font-size: 13px;
  }
  .link-actions .secondary:hover { border-color: #5eead4; }
  .footer { margin-top: 48px; color: #6f7a98; font-size: 12px; }
  .footer code { color: #93c5fd; }
  .panel {
    margin-top: 48px; border: 1px solid #243050; background: #101830;
    border-radius: 14px; padding: 22px; align-self: start;
  }
  .panel-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
  .panel h2 { font-size: 16px; margin: 4px 0 0; word-break: break-all; }
  .eyebrow { text-transform: uppercase; letter-spacing: 1.4px; font-size: 11px; color: #5eead4; }
  .icon-button {
    background: transparent; border: 1px solid #2b3758; color: #a3adc9;
    width: 30px; height: 30px; border-radius: 8px; cursor: pointer; font-size: 18px; line-height: 1;
  }
  .total { display: flex; align-items: baseline; gap: 10px; margin: 24px 0; }
  .total-number { font-size: 40px; font-weight: 800; color: #5eead4; }
  .total-label { color: #8b96b5; font-size: 13px; }
  .panel section { margin-top: 22px; }
  .panel h3 { font-size: 13px; text-transform: uppercase; letter-spacing: 1px; color: #8b96b5; margin: 0 0 12px; }
  .bar-row { display: grid; grid-template-columns: 74px 1fr 36px; gap: 8px; align-items: center; margin-bottom: 8px; font-size: 12px; }
  .bar-label { color: #cfd8ee; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .bar-track { height: 8px; background: #1b2542; border-radius: 999px; overflow: hidden; }
  .bar-fill { height: 100%; background: linear-gradient(90deg, #5eead4, #60a5fa); border-radius: 999px; }
  .bar-value { text-align: right; color: #8b96b5; }
  .footnote { margin-top: 26px; font-size: 11px; color: #6f7a98; line-height: 1.5; }
  @media (max-width: 980px) { .layout.with-panel { grid-template-columns: 1fr; } }
`
