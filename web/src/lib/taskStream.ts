/**
 * Live task log stream (image builds / package publishes).
 *
 * POST /api/v1/images/build and /api/v1/packages/publish both return
 * {ok, build_id} immediately and run the task in the background; the
 * gateway proxies ops-extension's SSE endpoint /api/v1/builds/{id}/stream
 * which replays existing log lines and streams new ones until the task
 * finishes (terminal `done` event carries state/image/error).
 */

export interface TaskStreamLine {
  stream: string
  line: string
}

export interface TaskStreamHandle {
  close(): void
}

export function openTaskStream(
  buildId: string,
  opts: {
    onLog: (lines: TaskStreamLine[]) => void
    onState: (state: string) => void
    onDone: (done: { state: string; image?: string; error?: string }) => void
    onError?: (message: string) => void
  },
): TaskStreamHandle {
  const es = new EventSource(
    `/api/v1/builds/${encodeURIComponent(buildId)}/stream`,
  )
  const parse = <T>(e: MessageEvent): T | null => {
    try {
      return JSON.parse(e.data) as T
    } catch {
      return null
    }
  }
  es.addEventListener('log', e => {
    const ln = parse<TaskStreamLine>(e as MessageEvent)
    if (ln) opts.onLog([ln])
  })
  es.addEventListener('state', e => {
    const st = parse<{ state: string }>(e as MessageEvent)
    if (st) opts.onState(st.state)
  })
  es.addEventListener('done', e => {
    const d = parse<{ state: string; image?: string; error?: string }>(
      e as MessageEvent,
    )
    es.close()
    opts.onDone(d ?? { state: 'unknown' })
  })
  es.onerror = () => {
    // The server closes the stream after `done`; a raw error without a
    // prior done event means the connection dropped mid-task.
    if (opts.onError) opts.onError('stream connection lost')
  }
  return { close: () => es.close() }
}
