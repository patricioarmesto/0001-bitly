# Códigos cortos base62 generados en el backend

Los códigos de 7 caracteres se generan con `crypto/rand` mapeado al alfabeto base62 y se insertan con `ON CONFLICT DO NOTHING`, reintentando con otro valor si hay colisión. Se descartaron Snowflake, UUID y una secuencia de Postgres: agregan un servicio coordinado, producen códigos más largos o predecibles, y el espacio de 62^7 cubre el ejercicio sin esa infraestructura.

**Consequences:** cambiar el alfabeto o el largo después de publicar dejaría sin resolver los links ya emitidos, así que el formato queda fijo.
