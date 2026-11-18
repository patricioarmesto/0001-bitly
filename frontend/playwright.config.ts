import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  expect: { timeout: 9_000 },
  retries: 0,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  // Playwright necesita un proceso que siga vivo. Sin `--wait` (que implica
  // detached), Compose queda en primer plano. Al terminar, Playwright corta el
  // CLI pero los contenedores siguen arriba, así que la próxima corrida reusa
  // el stack. El frontend depende del backend healthy, que a su vez depende de
  // Postgres, Redis, Kafka y ClickHouse: esperar a Vite alcanza.
  webServer: {
    command: 'docker compose -f ../docker-compose.yml up',
    url: 'http://localhost:5173',
    reuseExistingServer: true,
    timeout: 300_000,
  },
})
