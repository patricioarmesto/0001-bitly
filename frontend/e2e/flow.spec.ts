import { expect, test, type Page } from '@playwright/test'

const API_URL = 'http://localhost:8080'
// El redirect termina en el propio frontend: es una URL http válida y el test
// no depende de internet.
const LONG_URL = 'http://localhost:5173/'

async function shorten(page: Page, longUrl: string) {
  await page.goto('/')
  await page.getByLabel('URL a acortar').fill(longUrl)
  await page.getByRole('button', { name: 'Acortar' }).click()

  const card = page.locator('.link-card').first()
  await expect(card).toBeVisible()

  const shortUrl = (await card.getByRole('link').innerText()).trim()
  const code = shortUrl.slice(API_URL.length + 1)
  return { card, shortUrl, code }
}

test('shortens a URL and shows the link card', async ({ page }) => {
  const longUrl = 'https://example.com/e2e/acortar'
  const { card, code } = await shorten(page, longUrl)

  expect(code).toMatch(/^[0-9a-zA-Z]+$/)
  await expect(card.getByRole('link')).toHaveAttribute('href', `${API_URL}/${code}`)
  await expect(card).toContainText(longUrl)
})

test('redirects from the short link to the original URL', async ({ page }) => {
  const { shortUrl } = await shorten(page, LONG_URL)

  await page.goto(shortUrl)

  await expect(page).toHaveURL(LONG_URL)
})

test('records the click and shows it in the stats panel', async ({ page, request }) => {
  const { card, shortUrl, code } = await shorten(page, LONG_URL)

  const redirect = await request.get(shortUrl)
  expect(redirect.status()).toBe(200)
  expect(redirect.url()).toBe(LONG_URL)

  // El redirect publica el click en Kafka y el consumer lo escribe en
  // ClickHouse: hay que esperar a que la fila exista antes de abrir el panel,
  // que consulta las stats una sola vez al montarse.
  await expect
    .poll(
      async () => {
        const response = await request.get(`${API_URL}/api/stats/${code}`)
        const stats = (await response.json()) as { total: number }
        return stats.total
      },
      { timeout: 15_000, message: 'el consumer no escribió el click en ClickHouse' },
    )
    .toBe(1)

  await card.getByRole('button', { name: 'Ver stats →' }).click()

  const panel = page.locator('aside.panel')
  await expect(
    panel.getByRole('heading', { name: `Estadísticas de ${shortUrl}` }),
  ).toBeVisible()
  await expect(panel.locator('.total-number')).toHaveText('1')
  // El browser no manda CF-IPCountry, así que el click queda en XX.
  await expect(panel.getByText('XX', { exact: true })).toBeVisible()
})

test('shows the API error', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('URL a acortar').fill('ftp://example.com/archivo')
  await page.getByRole('button', { name: 'Acortar' }).click()

  await expect(page.getByText('la URL debe usar http o https')).toBeVisible()
})
