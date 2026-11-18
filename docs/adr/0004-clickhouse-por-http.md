# ClickHouse se consulta por HTTP con `net/http`

Las consultas viajan como query en el body con parámetros `param_*` y se decodifican desde `FORMAT JSON`. Se eligió esto antes que el cliente oficial para que las tres agregaciones queden visibles en `backend/clickhouse.go`, sin una dependencia ni una API intermedia que explicar en un ejercicio breve. El modelo es una tabla append-only `analytics.clicks` con `MergeTree` ordenada por `(code, clicked_at)`.

## Opciones consideradas

- **`net/http` + `FORMAT JSON` (elegido):** la query se lee donde se usa; cero dependencias nuevas.
- **Cliente oficial de ClickHouse:** más ergonómico en producción, pero agrega una capa que oscurece qué se está agregando.
