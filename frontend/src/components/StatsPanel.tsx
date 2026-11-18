import { useEffect, useState } from 'react'
import { requestJSON, type ShortLink } from '../api'
import BarList from './BarList'

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
  link: ShortLink
  onClose: () => void
}

export default function StatsPanel({ apiUrl, link, onClose }: StatsPanelProps) {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    const controller = new AbortController()
    setStats(null)
    setError('')

    requestJSON<Stats>(
      `${apiUrl}/api/stats/${encodeURIComponent(link.code)}`,
      { signal: controller.signal },
      'No se pudieron cargar las estadísticas',
    )
      .then(setStats)
      .catch((err: Error) => {
        if (err.name !== 'AbortError') {
          setError(err.message)
        }
      })

    return () => controller.abort()
  }, [apiUrl, link.code])

  const maxCountry = Math.max(1, ...(stats?.by_country.map((item) => item.clicks) ?? [1]))
  const maxDay = Math.max(1, ...(stats?.by_day.map((item) => item.clicks) ?? [1]))

  return (
    <aside className="panel">
      <div className="panel-head">
        <div>
          <div className="eyebrow">Analíticas</div>
          <h2>Estadísticas de {link.short_url}</h2>
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

          <BarList
            title="Por país"
            items={stats.by_country.map((item) => ({ label: item.country, clicks: item.clicks }))}
            max={maxCountry}
          />

          <BarList
            title="Por día"
            items={stats.by_day.map((item) => ({ label: item.day, clicks: item.clicks }))}
            max={maxDay}
          />

          <p className="footnote">
            Los eventos llegan a Kafka y los escribe un consumer en ClickHouse. Puede pasar un instante
            hasta que se reflejen.
          </p>
        </>
      )}
    </aside>
  )
}
