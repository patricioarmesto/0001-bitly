# React, Vite y TypeScript en el frontend

El frontend es una SPA chica con React, Vite y TypeScript. El chequeo de la UI es `npm run build` (`tsc --noEmit` y el bundle); el recorrido completo desde el browser lo cubre Playwright (ADR 0008).

## Contexto

La UI tiene tres responsabilidades: cargar URLs con un formulario, mostrar los links creados y abrir un panel con estadísticas. El Estado es local y no hay router ni sesión.

## Opciones consideradas

- **HTML y JavaScript sin framework**: menos dependencias, pero el panel de stats y la lista de links requieren manejar DOM a mano y el ejemplo deja de ser legible.
- **Next.js u otro framework con SSR/RSC**: resuelve un problema (render del servidor y rutas) que este prototipo no tiene y suma una capa de configuración.
- **React + Vite + TypeScript (elegido)**: componentes para el formulario, la tarjeta y el panel; tipado estático para las respuestas del API; Vite levanta el dev server con un `npm run dev` y expone `VITE_API_URL`.

## Consecuencias

- El tipado de `ShortLink` y `Stats` documenta el contrato del API sin un archivo de esquema separado.
- TypeScript corre en el build, así que un cambio de forma en la respuesta del backend rompe el build antes que la app en el browser.
- `@vitejs/plugin-react` existe para el JSX y el hot reload; es la pieza mínima que Vite necesita para este proyecto.
- No hay tests de componentes: el flujo de punta a punta lo cubre Playwright (ADR 0008) y el detalle de cada componente queda para la revisión manual.
