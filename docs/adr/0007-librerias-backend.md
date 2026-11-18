# Librerías del backend

El backend usa la biblioteca estándar para HTTP y JSON, y tres clientes externos: `sarama` para Kafka, `lib/pq` para Postgres y `go-redis` para Redis. `go-sqlmock` aparece sólo en tests.

## Contexto

El ejercicio quiere que se vea el recorrido de una request: stubs HTTP explícitos, SQL visible y publicación de eventos. Una librería que oculte esos pasos haría el ejemplo menos legible.

## Opciones consideradas

- **Cliente oficial de ClickHouse**: está descartado en el ADR 0004; la interfaz HTTP con `net/http` deja la query a la vista.
- **`pgx` en lugar de `lib/pq`**: más prestaciones y `database/sql` nativo, pero el prototipo usa `database/sql` con un driver simple y no necesita pool ni tipos propios.
- **`kafka-go` en lugar de `sarama`**: API más chica, pero `sarama` cubre productor sincrónico y consumer group con lo que ya está probado en el compose.
- **go-redis**: evita escribir un cliente RESP a mano y permite inyectar el contexto en cada operación de caché.
- **go-sqlmock**: verifica en tests que el handler hace el `INSERT` sin levantar Postgres; es dependencia de desarrollo, no de runtime.

## Consecuencias

- Hay tres dependencias de runtime en `go.mod` además de las transitivas; cada una resuelve un sistema del stack y no se usa para abstracciones extra.
- Los errores y reintentos de Kafka y ClickHouse se ven en el código, no quedan escondidos detrás de un cliente de más alto nivel.
- Agregar una librería nueva exige una razón documentada acá o en el ADR del área correspondiente.
