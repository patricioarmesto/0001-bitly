# IDs Snowflake convertidos a base62

La key del link corto es un ID Snowflake de 63 bits dentro de un `int64`, con el layout `[timestamp 41 bits | worker_id 10 bits | secuencia 12 bits]`. La fórmula es `compositeID = (timestamp << 22) | (worker_id << 12) | secuencia` y la key es `encodeBase62(uint64(compositeID))` con el alfabeto `0-9a-zA-Z`. Postgres guarda solamente `links(short_code, long_url, created_at)`.

## Contexto

Se descartaron tres alternativas: un autoincremental simple (predecible), una secuencia central en Postgres (pone a la base en el camino de cada escritura) y un prefijo por servidor configurable a mano que no distinguiera bien los casos de reloj y secuencia. Snowflake resuelve la unicidad con la combinación de tiempo, identificador de servidor y contador local:

- `timestamp = time.Now().UnixMilli() - 1704067200000`, milisegundos desde `2024-01-01T00:00:00Z`. Cabe en 41 bits y es la parte alta del ID.
- `worker_id`: entero de 0 a 1023 leído de la env var `WORKER_ID` (en local, `0`). Cada servidor tiene un valor único y nunca se repite.
- `secuencia`: contador en memoria de 0 a 4095, con mutex en `Generator`. Se reinicia en cada milisegundo nuevo. Si se llena dentro del mismo milisegundo, se espera al siguiente. Si el reloj retrocede, se espera a alcanzar el último timestamp emitido.
- Ejemplo: `timestamp = 63158400123`, `worker_id = 0`, `secuencia = 0` → `compositeID = 264905530269499392` → `key = "jzgKpHNqNO"`.

El redirect busca `WHERE short_code = <key>`: la key ya identifica al link y no hace falta descomponerla.

## Consecuencias

- No hay colisiones al agregar servidores: la unicidad viene de `timestamp + worker_id + secuencia`, y cada servidor tiene un `worker_id` distinto. La secuencia es local, así que no hace falta coordinación con Postgres ni contadores compartidos.
- No es predecible en el sentido de una secuencia global que empiece en 1; el timestamp desde 2024 encabeza el ID.
- Los IDs quedan ordenados por tiempo de creación porque el timestamp ocupa los bits altos. Esa propiedad es numérica: no se traduce al orden lexicográfico de las keys base62 con este alfabeto.
- Cada servidor debe configurar su `WORKER_ID` y no duplicarlo. Con 10 bits hay hasta 1024 servidores.
- Un solo proceso está limitado a 4096 links por milisegundo; pasado ese límite absorbe espera hasta el siguiente milisegundo.
- El reloj atrasado bloquea al generador hasta recuperar el último timestamp usado. Es preferible a emitir IDs repetidos: el generador sigue esperando, sin importar el tamaño del desfase.
- `int64` alcanza para los 63 bits del compuesto; `encodeBase62` sigue trabajando con `uint64` porque el corrimiento y el alfabeto no cambiaron.
