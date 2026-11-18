# Postgres es la fuente de verdad; Redis es la caché del camino caliente

Los links viven sólo en Postgres y Redis guarda `link:<code>` con TTL de una hora para resolver el `302` sin consultar la base. Ante un miss o un error de Redis se lee Postgres y se repuebla la caché. Se descartó usar Redis como único almacén: perdería los links ante un reinicio y haría invisible la separación entre dato durable y caché que el ejercicio quiere mostrar.

**Consecuencias:** el redirect depende de dos sistemas, pero el camino caliente es un `GET` en memoria más un `302`, lo que sostiene el objetivo de menos de 50 ms y 1000 redirects por segundo en local.
