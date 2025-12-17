async function api(path, opts) {
  const r = await fetch(path, opts);
  return r.json();
}

async function refreshStatus() {
  const s = await api('/api/status');
  document.getElementById('status').innerText = `HEAD: ${s.head || '(none)'} — Staged: ${s.staged}`;
}

async function refreshObjects() {
  const keys = await api('/api/objects');
  document.getElementById('objectsList').innerText = JSON.stringify(keys, null, 2);
}

async function refreshCommits() {
  const commits = await api('/api/commits');
  const el = document.getElementById('commitList');
  el.innerHTML = '';
  commits.forEach(c => {
    const d = document.createElement('div');
    d.style.padding = '6px';
    d.style.borderBottom = '1px solid rgba(255,255,255,0.02)';
    d.innerHTML = `<strong>${c.message}</strong><div style="font-size:12px;color:#9aa7bf">${new Date(c.timestamp*1000).toLocaleString()}</div>`;
    el.appendChild(d);
  });
}

async function addEntity() {
  const id = document.getElementById('entityId').value.trim();
  const label = document.getElementById('entityLabel').value.trim();
  const type = document.getElementById('entityType').value;
  if (!id || !label) return alert('id and label required');
  const urn = `urn:codex:entity/${id}`;
  const obj = { urn, type, labels: { en: label }, properties: {} };
  await api('/api/entity', { method: 'POST', body: JSON.stringify(obj) });
  await refreshObjects();
  await refreshStatus();
}

async function annotate() {
  const text = document.getElementById('textUrn').value.trim();
  const entity = document.getElementById('entityUrn').value.trim();
  const start = parseInt(document.getElementById('start').value || '0');
  const end = parseInt(document.getElementById('end').value || '0');
  if (!text || !entity) return alert('text and entity required');
  const obj = { text_urn: text, entity_urn: entity, start, end, certainty: 1.0 };
  await api('/api/annotate', { method: 'POST', body: JSON.stringify(obj) });
  await refreshObjects();
  await refreshStatus();
}

async function commit() {
  const msg = prompt('Commit message:');
  if (!msg) return;
  await api('/api/commit', { method: 'POST', body: JSON.stringify({ message: msg }) });
  await refreshStatus();
  await refreshCommits();
}

async function exportData() {
  const out = await api('/api/export');
  const blob = new Blob([JSON.stringify(out, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `codex-export-${Date.now()}.json`;
  document.body.appendChild(a);
  a.click();
  a.remove();
}

window.addEventListener('load', () => {
  refreshStatus();
  refreshObjects();
  refreshCommits();
  document.getElementById('addEntityBtn').addEventListener('click', addEntity);
  document.getElementById('annotateBtn').addEventListener('click', annotate);
  document.getElementById('commitBtn').addEventListener('click', commit);
  document.getElementById('exportBtn').addEventListener('click', exportData);
});
