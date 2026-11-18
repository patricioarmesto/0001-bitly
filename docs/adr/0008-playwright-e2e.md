# Playwright para el test end to end

El recorrido completo desde el browser —pegar una URL, obtener el link corto, seguir el `302` y ver el click reflejado en el panel— se prueba con Playwright contra el stack de Docker Compose. Es la única dependencia de desarrollo del frontend además de las del build.

## Contexto

`npm run build` verifica tipos y bundle, y los tests del backend mockean Postgres, Redis y ClickHouse con `sqlmock` y un cliente de Redis sin conexión. Ninguno de los dos recorre el camino real: `POST` → Postgres → Redis → `302` → Kafka → ClickHouse → stats. El panel además depende de un consumer asíncrono, así que la única forma de comprobar que un click aparece es esperar contra el stack completo.

## Opciones consideradas

- **Smoke test con `curl`**: no suma dependencias y muestra el recorrido de la request, pero no prueba React ni el panel, y hay que parsear JSON a mano.
- **Test E2E en Go con build tag**: usa sólo la stdlib y entra bien en CI, pero tampoco ejercita el browser ni los componentes.
- **Playwright (elegido)**: maneja el browser real, espera condiciones con reintento (`expect.poll`, `expect(...).toBeVisible()`) y levanta el stack con `webServer` y `docker compose up -d --wait`. El costo es una dependencia de desarrollo y la descarga del binario de Chromium.

## Consecuencias

- `npm run test:e2e` levanta Compose si no está arriba y lo deja corriendo; si el stack ya estaba, lo reutiliza. La primera corrida necesita `npx playwright install chromium`.
- Los tests crean links nuevos en cada corrida (la key Snowflake es única), así que no hace falta limpiar Postgres ni ClickHouse.
- El país se verifica como `XX`: el browser no manda `CF-IPCountry`, así que el test cubre el fallback del ADR 0005 en lugar de simular geolocalización.
- `e2e/` y `playwright.config.ts` están en el `tsconfig`, así que `npm run build` también los tipa.
- No hay tests de componentes: el suite cubre el flujo completo y el detalle de cada componente queda para el build y la revisión manual.
