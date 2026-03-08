const DEFAULT_API_BASE = 'http://localhost:8080/api/v1'
const API_BASE_STORAGE_KEY = 'acpApiBase'

function readStoredApiBase() {
  const stored = window.localStorage.getItem(API_BASE_STORAGE_KEY)
  return (stored || DEFAULT_API_BASE).replace(/\/$/, '')
}

export async function apiRequest(apiBase, path, options = {}) {
  const res = await fetch(`${apiBase}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  })

  let data = null
  try {
    data = await res.json()
  } catch {
    data = null
  }

  if (!res.ok) {
    const message = data?.error?.message || `${res.status} ${res.statusText}`
    throw new Error(message)
  }

  return data
}

export function getInitialApiBase() {
  return readStoredApiBase()
}

export function saveApiBase(next) {
  const normalized = next.trim().replace(/\/$/, '')
  window.localStorage.setItem(API_BASE_STORAGE_KEY, normalized)
  return normalized
}

export function loadDashboardSummary(apiBase) {
  return apiRequest(apiBase, '/dashboard/summary')
}

export function loadSessions(apiBase) {
  return apiRequest(apiBase, '/sessions?page=1&page_size=20')
}

export function loadSessionDetail(apiBase, sessionId) {
  return apiRequest(apiBase, `/sessions/${encodeURIComponent(sessionId)}`)
}

export function loadSessionTimeline(apiBase, sessionId) {
  return apiRequest(apiBase, `/sessions/${encodeURIComponent(sessionId)}/timeline?limit=100&offset=0`)
}

export function loadApprovals(apiBase) {
  return apiRequest(apiBase, '/approvals?status=pending&page=1&page_size=20')
}

export function decideApproval(apiBase, approvalId, decision) {
  return apiRequest(apiBase, `/approvals/${encodeURIComponent(approvalId)}/decision`, {
    method: 'POST',
    body: JSON.stringify({
      decision,
      approver_id: 'frontend.react',
      decision_comment: `Decision from React frontend: ${decision}`,
    }),
  })
}

export function loadPolicies(apiBase) {
  return apiRequest(apiBase, '/policies?page=1&page_size=20')
}

export function setPolicyEnabled(apiBase, policyId, enabled) {
  return apiRequest(
    apiBase,
    enabled
      ? `/policies/${encodeURIComponent(policyId)}/enable`
      : `/policies/${encodeURIComponent(policyId)}/disable`,
    { method: 'POST' },
  )
}

export function createWebSocket(apiBase, onMessage) {
  const httpBase = apiBase.replace(/\/api\/v1$/, '')
  const wsBase = httpBase.replace(/^http/, 'ws')
  const url = `${wsBase}/ws/events`

  let ws = null
  let reconnectTimer = null

  function connect() {
    ws = new WebSocket(url)
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        onMessage(data)
      } catch { /* ignore malformed messages */ }
    }
    ws.onclose = () => {
      reconnectTimer = setTimeout(connect, 3000)
    }
    ws.onerror = () => {
      ws.close()
    }
  }

  connect()

  return () => {
    clearTimeout(reconnectTimer)
    if (ws) ws.close()
  }
}
