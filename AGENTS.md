# AGENTS.md

Acortador de URLs de ejercicio: el valor está en que el recorrido de la request sea legible, no en que escale. `README.md` describe la arquitectura y el alcance; `docs/adr/` guarda cada decisión con sus alternativas descartadas. Abrí el ADR del área antes de cambiar generación de keys, storage, clicks, stats o país.

## Arranque y verificación

- Stack: `docker compose up`; esperá `curl http://localhost:8080/health` antes de probar el frontend en `:5173`.
- Backend: desde `backend/`, `gofmt -l .`, `go test ./...`, `go vet ./...` y `go build ./...`. Terminá el trabajo solo con los cuatro limpios.
- Frontend: desde `frontend/`, `npm run build` (incluye `tsc --noEmit`). El dev server lo levanta Compose con `VITE_API_URL=http://localhost:8080`.
- Para tests de IDs, inyectá `now` y `sleep` en `Generator` (`backend/idgen_test.go`); los tests del handler usan `sqlmock` para verificar el `INSERT`.

## Convenciones

- Español rioplatense en docs, comentarios y mensajes de usuario; tono directo.
- Commits `tipo(scope): descripción` en minúsculas, con scope `backend`, `frontend`, `docker` o `docs`. Commitear y pushear solo cuando el usuario lo pida.
- Preferí código directo y pocas abstracciones: si una opción hace el ejemplo más corto y más visible, gana. Una dependencia nueva necesita una razón que quede registrada en el ADR correspondiente.
- `WORKER_ID`, `worker` y `servidor` son los nombres del identificador de servidor, en código y en docs.

## Invariantes

- La key es Snowflake (`backend/idgen.go`): 41 bits de timestamp desde `2024-01-01T00:00:00Z`, 10 de `WORKER_ID` y 12 de secuencia en memoria con mutex. `encodeBase62` trabaja con `uint64` y el alfabeto `0-9a-zA-Z`. Detalle en el ADR 0001.
- Postgres no genera IDs: guarda `links(short_code, long_url, created_at)` y el arranque crea el esquema con `IF NOT EXISTS`. Esta es la primera versión: no hay migraciones de esquemas viejos ni secuencia heredada.
- El click sale por Kafka desde un goroutine y el redirect no espera a ClickHouse; las stats son queries HTTP a ClickHouse, no agregaciones calculadas en Go.
- El país del click viene de headers con fallback `XX`: no hay base GeoIP. Es decisión de alcance, documentada en el ADR 0005.
