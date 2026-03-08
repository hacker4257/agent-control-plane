(function () {
  const state = {
    apiBase: localStorage.getItem("acpApiBase") || "http://localhost:8080/api/v1",
  };

  const el = {
    apiBase: document.getElementById("apiBase"),
    saveApiBaseBtn: document.getElementById("saveApiBaseBtn"),
    refreshAllBtn: document.getElementById("refreshAllBtn"),

    refreshSummaryBtn: document.getElementById("refreshSummaryBtn"),
    refreshSessionsBtn: document.getElementById("refreshSessionsBtn"),
    refreshApprovalsBtn: document.getElementById("refreshApprovalsBtn"),
    refreshPoliciesBtn: document.getElementById("refreshPoliciesBtn"),

    summaryStatus: document.getElementById("summaryStatus"),
    sessionsStatus: document.getElementById("sessionsStatus"),
    approvalsStatus: document.getElementById("approvalsStatus"),
    policiesStatus: document.getElementById("policiesStatus"),

    sessionsCount: document.getElementById("sessionsCount"),
    pendingApprovalsCount: document.getElementById("pendingApprovalsCount"),
    blockedActionsCount: document.getElementById("blockedActionsCount"),
    policyHitsCount: document.getElementById("policyHitsCount"),

    sessionsBody: document.getElementById("sessionsBody"),
    approvalsBody: document.getElementById("approvalsBody"),
    policiesBody: document.getElementById("policiesBody"),
  };

  function setStatus(node, message, type) {
    node.textContent = message || "";
    node.classList.remove("error", "ok");
    if (type) node.classList.add(type);
  }

  function fmtTime(v) {
    if (!v) return "-";
    const d = new Date(v);
    if (Number.isNaN(d.getTime())) return String(v);
    return d.toLocaleString();
  }

  async function api(path, options) {
    const url = `${state.apiBase}${path}`;
    const res = await fetch(url, {
      headers: { "Content-Type": "application/json" },
      ...options,
    });
    let data = null;
    try {
      data = await res.json();
    } catch (_) {
      data = null;
    }
    if (!res.ok) {
      const msg = data?.error?.message || `${res.status} ${res.statusText}`;
      throw new Error(msg);
    }
    return data;
  }

  async function loadSummary() {
    setStatus(el.summaryStatus, "Loading...");
    try {
      const data = await api("/dashboard/summary");
      el.sessionsCount.textContent = String(data.sessions_count ?? 0);
      el.pendingApprovalsCount.textContent = String(data.pending_approvals_count ?? 0);
      el.blockedActionsCount.textContent = String(data.blocked_actions_count ?? 0);
      el.policyHitsCount.textContent = String(data.policy_hits_count ?? 0);
      setStatus(el.summaryStatus, "Updated", "ok");
    } catch (err) {
      setStatus(el.summaryStatus, err.message, "error");
    }
  }

  async function loadSessions() {
    setStatus(el.sessionsStatus, "Loading...");
    el.sessionsBody.innerHTML = "";
    try {
      const data = await api("/sessions?page=1&page_size=20");
      const items = data.items || [];
      if (!items.length) {
        el.sessionsBody.innerHTML = '<tr><td colspan="5" class="empty">No sessions</td></tr>';
      } else {
        const rows = items
          .map(
            (s) => `
              <tr>
                <td>${escapeHtml(s.session_id || "")}</td>
                <td><span class="tag">${escapeHtml(s.status || "-")}</span></td>
                <td>${escapeHtml(s.environment || "-")}</td>
                <td>${escapeHtml(String(s.risk_score ?? "-"))}</td>
                <td>${escapeHtml(fmtTime(s.updated_at))}</td>
              </tr>
            `
          )
          .join("");
        el.sessionsBody.innerHTML = rows;
      }
      setStatus(el.sessionsStatus, `Loaded ${items.length} sessions`, "ok");
    } catch (err) {
      setStatus(el.sessionsStatus, err.message, "error");
      el.sessionsBody.innerHTML = '<tr><td colspan="5" class="empty">Failed to load</td></tr>';
    }
  }

  async function loadApprovals() {
    setStatus(el.approvalsStatus, "Loading...");
    el.approvalsBody.innerHTML = "";
    try {
      const data = await api("/approvals?status=pending&page=1&page_size=20");
      const items = data.items || [];
      if (!items.length) {
        el.approvalsBody.innerHTML = '<tr><td colspan="6" class="empty">No pending approvals</td></tr>';
      } else {
        const rows = items
          .map(
            (a) => `
              <tr>
                <td>${escapeHtml(a.approval_id || "")}</td>
                <td>${escapeHtml(a.action || "-")}</td>
                <td>${escapeHtml(a.tool || "-")}</td>
                <td>${escapeHtml(a.resource || "-")}</td>
                <td>${escapeHtml(fmtTime(a.requested_at))}</td>
                <td>
                  <div class="actions">
                    <button data-approval-id="${escapeAttr(a.approval_id || "")}" data-decision="approve">Approve</button>
                    <button class="danger" data-approval-id="${escapeAttr(a.approval_id || "")}" data-decision="reject">Reject</button>
                  </div>
                </td>
              </tr>
            `
          )
          .join("");
        el.approvalsBody.innerHTML = rows;
      }
      setStatus(el.approvalsStatus, `Loaded ${items.length} pending approvals`, "ok");
    } catch (err) {
      setStatus(el.approvalsStatus, err.message, "error");
      el.approvalsBody.innerHTML = '<tr><td colspan="6" class="empty">Failed to load</td></tr>';
    }
  }

  async function loadPolicies() {
    setStatus(el.policiesStatus, "Loading...");
    el.policiesBody.innerHTML = "";
    try {
      const data = await api("/policies?page=1&page_size=20");
      const items = data.items || [];
      if (!items.length) {
        el.policiesBody.innerHTML = '<tr><td colspan="6" class="empty">No policies</td></tr>';
      } else {
        const rows = items
          .map((p) => {
            const enabled = !!p.enabled;
            return `
              <tr>
                <td>${escapeHtml(p.policy_id || "")}</td>
                <td>${escapeHtml(p.name || "-")}</td>
                <td><span class="tag ${decisionClass(p.decision)}">${escapeHtml(p.decision || "-")}</span></td>
                <td>${escapeHtml(String(p.priority ?? "-"))}</td>
                <td><span class="tag ${enabled ? "enabled" : "disabled"}">${enabled ? "enabled" : "disabled"}</span></td>
                <td>
                  ${enabled
                    ? `<button class="secondary" data-policy-id="${escapeAttr(p.policy_id || "")}" data-enable="false">Disable</button>`
                    : `<button data-policy-id="${escapeAttr(p.policy_id || "")}" data-enable="true">Enable</button>`}
                </td>
              </tr>
            `;
          })
          .join("");
        el.policiesBody.innerHTML = rows;
      }
      setStatus(el.policiesStatus, `Loaded ${items.length} policies`, "ok");
    } catch (err) {
      setStatus(el.policiesStatus, err.message, "error");
      el.policiesBody.innerHTML = '<tr><td colspan="6" class="empty">Failed to load</td></tr>';
    }
  }

  async function actOnApproval(approvalID, decision) {
    setStatus(el.approvalsStatus, `${decision} ${approvalID}...`);
    try {
      await api(`/approvals/${encodeURIComponent(approvalID)}/decision`, {
        method: "POST",
        body: JSON.stringify({
          decision,
          approver_id: "frontend.demo",
          decision_comment: `Decision from frontend: ${decision}`,
        }),
      });
      setStatus(el.approvalsStatus, `Applied ${decision} on ${approvalID}`, "ok");
      await Promise.all([loadApprovals(), loadSummary()]);
    } catch (err) {
      setStatus(el.approvalsStatus, err.message, "error");
    }
  }

  async function setPolicyEnabled(policyID, enabled) {
    setStatus(el.policiesStatus, `${enabled ? "Enabling" : "Disabling"} ${policyID}...`);
    try {
      const path = enabled
        ? `/policies/${encodeURIComponent(policyID)}/enable`
        : `/policies/${encodeURIComponent(policyID)}/disable`;
      await api(path, { method: "POST" });
      setStatus(el.policiesStatus, `${enabled ? "Enabled" : "Disabled"} ${policyID}`, "ok");
      await Promise.all([loadPolicies(), loadSummary()]);
    } catch (err) {
      setStatus(el.policiesStatus, err.message, "error");
    }
  }

  async function refreshAll() {
    await Promise.all([loadSummary(), loadSessions(), loadApprovals(), loadPolicies()]);
  }

  function decisionClass(v) {
    const d = String(v || "").toLowerCase();
    if (d === "allow") return "allow";
    if (d === "block") return "block";
    if (d === "require_approval") return "pending";
    return "";
  }

  function escapeHtml(v) {
    return String(v)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function escapeAttr(v) {
    return escapeHtml(v);
  }

  function bindEvents() {
    el.saveApiBaseBtn.addEventListener("click", () => {
      const next = el.apiBase.value.trim().replace(/\/$/, "");
      if (!next) return;
      state.apiBase = next;
      localStorage.setItem("acpApiBase", state.apiBase);
      refreshAll();
    });

    el.refreshAllBtn.addEventListener("click", refreshAll);
    el.refreshSummaryBtn.addEventListener("click", loadSummary);
    el.refreshSessionsBtn.addEventListener("click", loadSessions);
    el.refreshApprovalsBtn.addEventListener("click", loadApprovals);
    el.refreshPoliciesBtn.addEventListener("click", loadPolicies);

    el.approvalsBody.addEventListener("click", (e) => {
      const btn = e.target.closest("button[data-approval-id]");
      if (!btn) return;
      actOnApproval(btn.getAttribute("data-approval-id"), btn.getAttribute("data-decision"));
    });

    el.policiesBody.addEventListener("click", (e) => {
      const btn = e.target.closest("button[data-policy-id]");
      if (!btn) return;
      setPolicyEnabled(btn.getAttribute("data-policy-id"), btn.getAttribute("data-enable") === "true");
    });
  }

  function init() {
    el.apiBase.value = state.apiBase;
    bindEvents();
    refreshAll();
  }

  init();
})();
