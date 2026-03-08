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
  if (
    v === 'allow' ||
    v === 'approved' ||
    v === 'enabled' ||
    v === 'tool_requested' ||
    v === 'tool_completed' ||
    v === 'completed'
  ) return 'ok'
  if (
    v === 'block' ||
    v === 'rejected' ||
    v === 'disabled' ||
    v === 'policy_blocked' ||
    v === 'tool_failed' ||
    v === 'blocked'
  ) return 'danger'
  if (v === 'require_approval' || v === 'pending' || v === 'approval_requested' || v === 'approval_pending') return 'warn'
  return 'neutral'
}

function humanizeEventType(eventType) {
  switch (eventType) {
    case 'tool_requested':
      return 'Tool Requested'
    case 'policy_blocked':
      return 'Policy Blocked'
    case 'approval_requested':
      return 'Approval Requested'
    case 'tool_completed':
      return 'Tool Completed'
    case 'tool_failed':
      return 'Tool Failed'
    default:
      return eventType || '-'
  }
}

function eventSummary(event) {
  switch (event.event_type) {
    case 'tool_requested':
      return `${event.tool || 'tool'} requested ${event.action || 'action'}`
    case 'policy_blocked':
      return `${event.tool || 'tool'} was blocked by policy`
    case 'approval_requested':
      return `${event.tool || 'tool'} requires human approval`
    case 'tool_completed':
      return `${event.tool || 'tool'} completed successfully`
    case 'tool_failed':
      return `${event.tool || 'tool'} failed during execution`
    default:
      return event.reason_text || event.input_summary || event.output_summary || 'No summary'
  }
}

function renderList(values) {
  if (!values || !values.length) return '-'
  return values.join(', ')
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

function SessionOverview({ session }) {
  if (!session) {
    return <div className="empty">Select a session to inspect the agent behavior.</div>
  }

  return (
    <div className="detail-panel session-overview-panel">
      <h3>Session Overview</h3>
      <dl className="detail-grid">
        <dt>Session ID</dt><dd>{session.session_id}</dd>
        <dt>Agent</dt><dd>{session.agent_id || '-'}</dd>
        <dt>User</dt><dd>{session.user_id || '-'}</dd>
        <dt>Status</dt><dd><span className={`tag ${decisionClass(session.status)}`}>{session.status || '-'}</span></dd>
        <dt>Environment</dt><dd>{session.environment || '-'}</dd>
        <dt>Objective</dt><dd>{session.objective || '-'}</dd>
        <dt>Risk Score</dt><dd>{session.risk_score ?? '-'}</dd>
        <dt>Approvals</dt><dd>{session.approvals_count ?? 0}</dd>
        <dt>Blocked</dt><dd>{session.blocked_count ?? 0}</dd>
        <dt>Resources</dt><dd>{renderList(session.touched_resources)}</dd>
        <dt>Last Event</dt><dd>{fmtTime(session.last_event_at)}</dd>
      </dl>
    </div>
  )
}

function TimelineList({ timeline, selectedEventId, onSelect }) {
  if (!timeline.length) {
    return <div className="empty">No timeline events</div>
  }

  return (
    <ul className="timeline-list">
      {timeline.map((event) => (
        <li
          key={event.event_id}
          className={`timeline-item ${selectedEventId === event.event_id ? 'timeline-item-selected' : ''}`}
          onClick={() => onSelect(event.event_id)}
        >
          <div className="timeline-meta">
            <span className={`tag ${decisionClass(event.event_type || event.decision)}`}>
              {humanizeEventType(event.event_type)}
            </span>
            <span>{fmtTime(event.created_at)}</span>
          </div>
          <div className="timeline-main">
            <strong>{event.tool || event.action || 'event'}</strong>
            <p>{eventSummary(event)}</p>
            <small>
              action: {event.action || '-'} · resource: {event.resource || '-'} · actor: {event.actor_id || '-'}
            </small>
          </div>
        </li>
      ))}
    </ul>
  )
}

function EventDetail({ event }) {
  if (!event) {
    return <div className="empty">Select a timeline event to inspect details.</div>
  }

  return (
    <div className="detail-panel event-detail-panel">
      <h3>Event Detail</h3>
      <div className="event-headline">
        <span className={`tag ${decisionClass(event.event_type || event.decision)}`}>{humanizeEventType(event.event_type)}</span>
        <strong>{eventSummary(event)}</strong>
      </div>
      <dl className="detail-grid">
        <dt>Event ID</dt><dd>{event.event_id}</dd>
        <dt>Session ID</dt><dd>{event.session_id || '-'}</dd>
        <dt>Step ID</dt><dd>{event.step_id || '-'}</dd>
        <dt>Correlation ID</dt><dd>{event.correlation_id || '-'}</dd>
        <dt>Tool</dt><dd>{event.tool || '-'}</dd>
        <dt>Action</dt><dd>{event.action || '-'}</dd>
        <dt>Resource</dt><dd>{event.resource || '-'}</dd>
        <dt>Decision</dt><dd>{event.decision || '-'}</dd>
        <dt>Reason Code</dt><dd>{event.reason_code || '-'}</dd>
        <dt>Reason Text</dt><dd>{event.reason_text || '-'}</dd>
        <dt>Risk Score</dt><dd>{event.risk_score ?? '-'}</dd>
        <dt>Risk Tags</dt><dd>{renderList(event.risk_tags)}</dd>
        <dt>Policies</dt><dd>{renderList(event.matched_policy_ids)}</dd>
        <dt>Input Summary</dt><dd>{event.input_summary || '-'}</dd>
        <dt>Output Summary</dt><dd>{event.output_summary || '-'}</dd>
        <dt>Artifacts</dt><dd>{renderList(event.artifact_refs)}</dd>
        <dt>Actor Type</dt><dd>{event.actor_type || '-'}</dd>
        <dt>Actor ID</dt><dd>{event.actor_id || '-'}</dd>
        <dt>Created At</dt><dd>{fmtTime(event.created_at)}</dd>
      </dl>
    </div>
  )
}

function ObservabilityPanel({ session, timeline, selectedEventId, onSelectEvent }) {
  const selectedEvent = useMemo(
    () => timeline.find((item) => item.event_id === selectedEventId) || null,
    [timeline, selectedEventId],
  )

  return (
    <div className="observability-layout">
      <SessionOverview session={session} />
      <div className="detail-panel trace-panel">
        <h3>Behavior Timeline</h3>
        <TimelineList timeline={timeline} selectedEventId={selectedEventId} onSelect={onSelectEvent} />
      </div>
      <EventDetail event={selectedEvent} />
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
            <th>Reason</th>
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
              <td>{approval.trigger_reason || '-'}</td>
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
  const [selectedEventId, setSelectedEventId] = useState('')
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
      setSelectedEventId('')
      return
    }

    setDetailStatus({ message: 'Loading session trace...' })
    try {
      const [detail, timelineData] = await Promise.all([
        loadSessionDetail(apiBase, sessionId),
        loadSessionTimeline(apiBase, sessionId),
      ])
      const items = timelineData.items || []
      setSessionDetail(detail)
      setTimeline(items)
      setSelectedEventId(items[items.length - 1]?.event_id || '')
      setDetailStatus({ message: `Loaded ${sessionId}`, type: 'ok' })
    } catch (error) {
      setSessionDetail(null)
      setTimeline([])
      setSelectedEventId('')
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
      await Promise.all([refreshApprovals(), refreshSummary(), refreshSessionDetail(selectedSessionId)])
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
          <p className="subtitle">Gateway observability console for external agent behavior</p>
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
          title={selectedSession ? `Observability Trace · ${selectedSession.session_id}` : 'Observability Trace'}
          status={detailStatus}
          actions={selectedSessionId ? <button onClick={() => refreshSessionDetail(selectedSessionId)}>Refresh</button> : null}
        >
          <ObservabilityPanel
            session={sessionDetail || selectedSession}
            timeline={timeline}
            selectedEventId={selectedEventId}
            onSelectEvent={setSelectedEventId}
          />
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
