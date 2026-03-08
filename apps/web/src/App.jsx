import { useEffect, useMemo, useState } from 'react'
import {
  decideApproval,
  getInitialApiBase,
  loadApprovals,
  loadDashboardSummary,
  loadPolicies,
  loadSessionDetail,
  loadSessions,
  loadSessionTimeline,
  saveApiBase,
  setPolicyEnabled,
} from './api'

function fmtTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString()
}

function decisionClass(value) {
  const v = String(value || '').toLowerCase()
  if (v === 'allow' || v === 'approved' || v === 'enabled') return 'ok'
  if (v === 'block' || v === 'rejected' || v === 'disabled') return 'danger'
  if (v === 'require_approval' || v === 'pending') return 'warn'
  return 'neutral'
}

function SectionCard({ title, actions, status, children }) {
  return (
    <section className="card">
      <div className="card-header">
        <h2>{title}</h2>
        <div className="toolbar">{actions}</div>
      </div>
      <p className={`status ${status.type || ''}`}>{status.message || ''}</p>
      {children}
    </section>
  )
}

function SummaryGrid({ summary }) {
  const items = [
    ['Sessions', summary.sessions_count ?? 0],
    ['Pending Approvals', summary.pending_approvals_count ?? 0],
    ['Blocked Actions', summary.blocked_actions_count ?? 0],
    ['Policy Hits', summary.policy_hits_count ?? 0],
  ]

  return (
    <div className="summary-grid">
      {items.map(([label, value]) => (
        <div key={label} className="summary-item">
          <span>{label}</span>
          <strong>{value}</strong>
        </div>
      ))}
    </div>
  )
}

function SessionTable({ sessions, selectedSessionId, onSelect }) {
  if (!sessions.length) {
    return <div className="empty">No sessions</div>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Session ID</th>
            <th>Status</th>
            <th>Environment</th>
            <th>Risk</th>
            <th>Updated</th>
          </tr>
        </thead>
        <tbody>
          {sessions.map((session) => (
            <tr
              key={session.session_id}
              className={selectedSessionId === session.session_id ? 'selected-row' : ''}
              onClick={() => onSelect(session.session_id)}
            >
              <td>{session.session_id}</td>
              <td><span className={`tag ${decisionClass(session.status)}`}>{session.status || '-'}</span></td>
              <td>{session.environment || '-'}</td>
              <td>{session.risk_score ?? '-'}</td>
              <td>{fmtTime(session.updated_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function SessionDetail({ session, timeline }) {
  if (!session) {
    return <div className="empty">Select a session to view details and timeline.</div>
  }

  return (
    <div className="detail-layout">
      <div className="detail-panel">
        <h3>Session Detail</h3>
        <dl className="detail-grid">
          <dt>Session ID</dt><dd>{session.session_id}</dd>
          <dt>Agent</dt><dd>{session.agent_id || '-'}</dd>
          <dt>User</dt><dd>{session.user_id || '-'}</dd>
          <dt>Status</dt><dd>{session.status || '-'}</dd>
          <dt>Environment</dt><dd>{session.environment || '-'}</dd>
          <dt>Objective</dt><dd>{session.objective || '-'}</dd>
          <dt>Risk Score</dt><dd>{session.risk_score ?? '-'}</dd>
          <dt>Approvals</dt><dd>{session.approvals_count ?? 0}</dd>
          <dt>Blocked</dt><dd>{session.blocked_count ?? 0}</dd>
          <dt>Last Event</dt><dd>{fmtTime(session.last_event_at)}</dd>
        </dl>
      </div>
      <div className="detail-panel">
        <h3>Timeline</h3>
        {!timeline.length ? (
          <div className="empty">No timeline events</div>
        ) : (
          <ul className="timeline-list">
            {timeline.map((event) => (
              <li key={event.event_id} className="timeline-item">
                <div className="timeline-meta">
                  <span className={`tag ${decisionClass(event.decision || event.event_type)}`}>
                    {event.event_type || '-'}
                  </span>
                  <span>{fmtTime(event.created_at)}</span>
                </div>
                <div className="timeline-main">
                  <strong>{event.tool || event.action || 'event'}</strong>
                  <p>{event.reason_text || event.input_summary || '-'}</p>
                  <small>
                    resource: {event.resource || '-'} · decision: {event.decision || '-'} · actor: {event.actor_id || '-'}
                  </small>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function ApprovalsTable({ approvals, onDecision, busyId }) {
  if (!approvals.length) {
    return <div className="empty">No pending approvals</div>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Approval ID</th>
            <th>Action</th>
            <th>Tool</th>
            <th>Resource</th>
            <th>Requested At</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {approvals.map((approval) => (
            <tr key={approval.approval_id}>
              <td>{approval.approval_id}</td>
              <td>{approval.action || '-'}</td>
              <td>{approval.tool || '-'}</td>
              <td>{approval.resource || '-'}</td>
              <td>{fmtTime(approval.requested_at)}</td>
              <td>
                <div className="actions">
                  <button disabled={busyId === approval.approval_id} onClick={() => onDecision(approval.approval_id, 'approve')}>
                    Approve
                  </button>
                  <button className="danger" disabled={busyId === approval.approval_id} onClick={() => onDecision(approval.approval_id, 'reject')}>
                    Reject
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function PoliciesTable({ policies, onToggle, busyId }) {
  if (!policies.length) {
    return <div className="empty">No policies</div>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Policy ID</th>
            <th>Name</th>
            <th>Decision</th>
            <th>Priority</th>
            <th>Enabled</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {policies.map((policy) => (
            <tr key={policy.policy_id}>
              <td>{policy.policy_id}</td>
              <td>{policy.name || '-'}</td>
              <td><span className={`tag ${decisionClass(policy.decision)}`}>{policy.decision || '-'}</span></td>
              <td>{policy.priority ?? '-'}</td>
              <td><span className={`tag ${policy.enabled ? 'ok' : 'danger'}`}>{policy.enabled ? 'enabled' : 'disabled'}</span></td>
              <td>
                <button
                  className={policy.enabled ? 'secondary' : ''}
                  disabled={busyId === policy.policy_id}
                  onClick={() => onToggle(policy.policy_id, !policy.enabled)}
                >
                  {policy.enabled ? 'Disable' : 'Enable'}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function App() {
  const [apiBaseInput, setApiBaseInput] = useState(getInitialApiBase())
  const [apiBase, setApiBase] = useState(getInitialApiBase())

  const [summary, setSummary] = useState({})
  const [sessions, setSessions] = useState([])
  const [selectedSessionId, setSelectedSessionId] = useState('')
  const [sessionDetail, setSessionDetail] = useState(null)
  const [timeline, setTimeline] = useState([])
  const [approvals, setApprovals] = useState([])
  const [policies, setPolicies] = useState([])

  const [summaryStatus, setSummaryStatus] = useState({ message: '' })
  const [sessionsStatus, setSessionsStatus] = useState({ message: '' })
  const [detailStatus, setDetailStatus] = useState({ message: '' })
  const [approvalsStatus, setApprovalsStatus] = useState({ message: '' })
  const [policiesStatus, setPoliciesStatus] = useState({ message: '' })

  const [approvalBusyId, setApprovalBusyId] = useState('')
  const [policyBusyId, setPolicyBusyId] = useState('')

  async function refreshSummary() {
    setSummaryStatus({ message: 'Loading...' })
    try {
      const data = await loadDashboardSummary(apiBase)
      setSummary(data)
      setSummaryStatus({ message: 'Updated', type: 'ok' })
    } catch (error) {
      setSummaryStatus({ message: error.message, type: 'error' })
    }
  }

  async function refreshSessions() {
    setSessionsStatus({ message: 'Loading...' })
    try {
      const data = await loadSessions(apiBase)
      const items = data.items || []
      setSessions(items)
      setSessionsStatus({ message: `Loaded ${items.length} sessions`, type: 'ok' })
      if (!selectedSessionId && items[0]?.session_id) {
        setSelectedSessionId(items[0].session_id)
      }
      if (selectedSessionId && !items.some((item) => item.session_id === selectedSessionId)) {
        setSelectedSessionId(items[0]?.session_id || '')
      }
    } catch (error) {
      setSessions([])
      setSessionsStatus({ message: error.message, type: 'error' })
    }
  }

  async function refreshSessionDetail(sessionId) {
    if (!sessionId) {
      setSessionDetail(null)
      setTimeline([])
      return
    }

    setDetailStatus({ message: 'Loading session detail...' })
    try {
      const [detail, timelineData] = await Promise.all([
        loadSessionDetail(apiBase, sessionId),
        loadSessionTimeline(apiBase, sessionId),
      ])
      setSessionDetail(detail)
      setTimeline(timelineData.items || [])
      setDetailStatus({ message: `Loaded ${sessionId}`, type: 'ok' })
    } catch (error) {
      setSessionDetail(null)
      setTimeline([])
      setDetailStatus({ message: error.message, type: 'error' })
    }
  }

  async function refreshApprovals() {
    setApprovalsStatus({ message: 'Loading...' })
    try {
      const data = await loadApprovals(apiBase)
      const items = data.items || []
      setApprovals(items)
      setApprovalsStatus({ message: `Loaded ${items.length} pending approvals`, type: 'ok' })
    } catch (error) {
      setApprovals([])
      setApprovalsStatus({ message: error.message, type: 'error' })
    }
  }

  async function refreshPolicies() {
    setPoliciesStatus({ message: 'Loading...' })
    try {
      const data = await loadPolicies(apiBase)
      const items = data.items || []
      setPolicies(items)
      setPoliciesStatus({ message: `Loaded ${items.length} policies`, type: 'ok' })
    } catch (error) {
      setPolicies([])
      setPoliciesStatus({ message: error.message, type: 'error' })
    }
  }

  async function refreshAll() {
    await Promise.all([refreshSummary(), refreshSessions(), refreshApprovals(), refreshPolicies()])
  }

  async function handleApprovalDecision(approvalId, decision) {
    setApprovalBusyId(approvalId)
    setApprovalsStatus({ message: `${decision} ${approvalId}...` })
    try {
      await decideApproval(apiBase, approvalId, decision)
      setApprovalsStatus({ message: `Applied ${decision} on ${approvalId}`, type: 'ok' })
      await Promise.all([refreshApprovals(), refreshSummary()])
    } catch (error) {
      setApprovalsStatus({ message: error.message, type: 'error' })
    } finally {
      setApprovalBusyId('')
    }
  }

  async function handlePolicyToggle(policyId, enabled) {
    setPolicyBusyId(policyId)
    setPoliciesStatus({ message: `${enabled ? 'Enabling' : 'Disabling'} ${policyId}...` })
    try {
      await setPolicyEnabled(apiBase, policyId, enabled)
      setPoliciesStatus({ message: `${enabled ? 'Enabled' : 'Disabled'} ${policyId}`, type: 'ok' })
      await Promise.all([refreshPolicies(), refreshSummary()])
    } catch (error) {
      setPoliciesStatus({ message: error.message, type: 'error' })
    } finally {
      setPolicyBusyId('')
    }
  }

  function handleSaveApiBase() {
    const normalized = saveApiBase(apiBaseInput)
    setApiBase(normalized)
  }

  useEffect(() => {
    refreshAll()
  }, [apiBase])

  useEffect(() => {
    refreshSessionDetail(selectedSessionId)
  }, [apiBase, selectedSessionId])

  const selectedSession = useMemo(
    () => sessions.find((item) => item.session_id === selectedSessionId) || null,
    [sessions, selectedSessionId],
  )

  return (
    <div className="app-shell">
      <header className="topbar">
        <div>
          <h1>Agent Control Plane</h1>
          <p className="subtitle">React control console for ACP MVP</p>
        </div>
        <div className="api-config">
          <label htmlFor="apiBase">API Base</label>
          <input
            id="apiBase"
            type="text"
            value={apiBaseInput}
            onChange={(event) => setApiBaseInput(event.target.value)}
          />
          <button onClick={handleSaveApiBase}>Save</button>
          <button className="secondary" onClick={refreshAll}>Refresh</button>
        </div>
      </header>

      <main className="grid">
        <SectionCard
          title="Dashboard Summary"
          status={summaryStatus}
          actions={<button onClick={refreshSummary}>Refresh</button>}
        >
          <SummaryGrid summary={summary} />
        </SectionCard>

        <SectionCard
          title="Sessions"
          status={sessionsStatus}
          actions={<button onClick={refreshSessions}>Refresh</button>}
        >
          <SessionTable
            sessions={sessions}
            selectedSessionId={selectedSessionId}
            onSelect={setSelectedSessionId}
          />
        </SectionCard>

        <SectionCard
          title={selectedSession ? `Session Detail · ${selectedSession.session_id}` : 'Session Detail & Timeline'}
          status={detailStatus}
          actions={selectedSessionId ? <button onClick={() => refreshSessionDetail(selectedSessionId)}>Refresh</button> : null}
        >
          <SessionDetail session={sessionDetail || selectedSession} timeline={timeline} />
        </SectionCard>

        <SectionCard
          title="Pending Approvals"
          status={approvalsStatus}
          actions={<button onClick={refreshApprovals}>Refresh</button>}
        >
          <ApprovalsTable approvals={approvals} onDecision={handleApprovalDecision} busyId={approvalBusyId} />
        </SectionCard>

        <SectionCard
          title="Policies"
          status={policiesStatus}
          actions={<button onClick={refreshPolicies}>Refresh</button>}
        >
          <PoliciesTable policies={policies} onToggle={handlePolicyToggle} busyId={policyBusyId} />
        </SectionCard>
      </main>
    </div>
  )
}
