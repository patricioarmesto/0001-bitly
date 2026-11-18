import { useEffect, useState } from 'react'

export interface CountryStat {
  country: string
  clicks: number
}

export interface DayStat {
  day: string
  clicks: number
}

export interface Stats {
  code: string
  total: number
  by_country: CountryStat[]
  by_day: DayStat[]
}

interface StatsPanelProps {
  apiUrl: string
  code: string
  shortUrl: string
  onClose: () => void
}

export default function StatsPanel({ apiUrl, code, shortUrl, onClose }: StatsPanelProps) {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    const controller = new AbortController()
    setStats(null)
    setError('')

    fetch(`${apiUrl}/api/stats/${encodeURIComponent(code)}`, { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error('No se pudieron cargar las estadísticas')
        }
        return (await response.json()) as Stats
      })
      .then(setStats)
      .catch((err: Error) => {
        if (err.name !== 'AbortError') {
          setError(err.message)
        }
      })

    return () => controller.abort()
  }, [apiUrl, code])

  const maxCountry = Math.max(1, ...(stats?.by_country.map((item) => item.clicks) ?? [1]))
  const maxDay = Math.max(1, ...(stats?.by_day.map((item) => item.clicks) ?? [1]))

  return (
    <aside className="panel">
      <div className="panel-head">
        <div>
          <div className="eyebrow">Analíticas</div>
          <h2>Estadísticas de {shortUrl}</h2>
        </div>
        <button className="icon-button" onClick={onClose} aria-label="Cerrar panel">
          ×
        </button>
      </div>

      {error && <p className="error">{error}</p>}

      {!error && !stats && <p className="muted">Consultando ClickHouse…</p>}

      {stats && (
        <>
          <div className="total">
            <span className="total-number">{stats.total.toLocaleString('es-AR')}</span>
            <span className="total-label">clicks registrados</span>
          </div>

          <section>
            <h3>Por país</h3>
            {stats.by_country.length === 0 && <p className="muted">Todavía no hay clicks.</p>}
            {stats.by_country.map((item) => (
              <div className="bar-row" key={item.country}>
                <span className="bar-label">{item.country}</span>
                <div className="bar-track">
                  <div className="bar-fill" style={{ width: `${(item.clicks / maxCountry) * 100}%` }} />
                </div>
                <span className="bar-value">{item.clicks}</span>
              </div>
            ))}
          </section>

          <section>
            <h3>Por día</h3>
            {stats.by_day.length === 0 && <p className="muted">Todavía no hay clicks.</p>}
            {stats.by_day.map((item) => (
              <div className="bar-row" key={item.day}>
                <span className="bar-label">{item.day}</span>
                <div className="bar-track">
                  <div className="bar-fill" style={{ width: `${(item.clicks / maxDay) * 100}%` }} />
                </div>
                <span className="bar-value">{item.clicks}</span>
              </div>
            ))}
          </section>

          <p className="footnote">
            Los eventos llegan a Kafka y los escribe un consumer en ClickHouse. Puede pasar un instante
            hasta que se reflejen.
          </p>
        </>
      )}
    </aside>
  )
}
