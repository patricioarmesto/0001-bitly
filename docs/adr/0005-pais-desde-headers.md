# El país del click se lee de headers, sin GeoIP

Se toma `CF-IPCountry` o `X-Country`, con fallback `XX`. Resolver la IP con una base GeoIP agregaría un archivo de datos o una llamada externa al camino caliente sin aportar nada al mecanismo de acortamiento, caché y eventos que se quiere explicar.

**Consequences:** los clicks sin esas headers se agrupan bajo `XX`; el panel por país muestra lo que informa el edge o el cliente, no una geolocalización propia.
