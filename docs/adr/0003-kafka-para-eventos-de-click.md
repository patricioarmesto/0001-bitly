# Los clicks viajan por Kafka, fuera del camino del redirect

Cada redirect encola `{code, country, clicked_at}` en el topic `clicks` desde un goroutine, con reintento simple y backoff, y un consumer group lo escribe en ClickHouse. Escribir directamente a ClickHouse desde el handler acoplaría el latency de analíticas al redirect y sacaría el bus de eventos que se está estudiando. La entrega es at-least-once: el offset se commitea después del insert, así que un replay puede duplicar filas.

## Considered Options

- **Kafka + consumer group (elegido):** desacopla el redirect de las analíticas y hace explícito el punto de entrega.
- **INSERT directo desde el handler:** menos piezas, pero mete la escritura analítica en el camino caliente.

## Consequences

Los duplicados ocasionales son aceptables para un contador de clicks. Reintentos complejos, DLT, idempotencia, exactly-once y Transactional Outbox quedan fuera de alcance a propósito; las razones están en «Por qué esto es simple» del README.
