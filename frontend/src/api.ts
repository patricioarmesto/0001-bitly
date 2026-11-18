export interface ShortLink {
  code: string
  short_url: string
  long_url: string
}

export async function requestJSON<T>(
  url: string,
  options: RequestInit = {},
  fallbackError = 'No se pudo completar la solicitud',
): Promise<T> {
  const response = await fetch(url, options)
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(body?.error ?? fallbackError)
  }
  return (await response.json()) as T
}
