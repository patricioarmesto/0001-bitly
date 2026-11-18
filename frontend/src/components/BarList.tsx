interface BarItem {
  label: string
  clicks: number
}

interface BarListProps {
  title: string
  items: BarItem[]
  max: number
}

export default function BarList({ title, items, max }: BarListProps) {
  return (
    <section>
      <h3>{title}</h3>
      {items.length === 0 && <p className="muted">Todavía no hay clicks.</p>}
      {items.map((item) => (
        <div className="bar-row" key={item.label}>
          <span className="bar-label">{item.label}</span>
          <div className="bar-track">
            <div className="bar-fill" style={{ width: `${(item.clicks / max) * 100}%` }} />
          </div>
          <span className="bar-value">{item.clicks}</span>
        </div>
      ))}
    </section>
  )
}
