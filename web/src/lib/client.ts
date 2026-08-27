/**
 * Path builders for the gateway API. The legacy hono-RPC client was
 * replaced by plain paths against the gateway-go aggregate surface; response
 * shapes stay validated by the zod schemas at each call site.
 */

const origin =
  typeof window !== 'undefined' ? window.location.origin : 'http://localhost'

export { origin }

export function qs(params: Record<string, string | number | undefined>): string {
  const sp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') sp.set(k, String(v))
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}

/** GET `/api/v1/sessions/:id/stream` as an SSE stream. */
export function sessionEventsUrl(sessionId: string): string {
  return `/api/v1/sessions/${encodeURIComponent(sessionId)}/stream`
}
