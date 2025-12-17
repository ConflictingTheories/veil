async function api(path, opts = {}) {
  const res = await fetch(path, opts);
  if (!res.ok) throw new Error(`API ${path} failed: ${res.status}`);
  return res.json();
}

function setOut(v) { document.getElementById('output').textContent = typeof v === 'string' ? v : JSON.stringify(v, null, 2) }

document.getElementById('btnStatus').addEventListener('click', async () => {
  try { setOut('loading...'); const s = await api('/api/codex/status'); setOut(s) } catch (e) { setOut(e.message) }
})

document.getElementById('btnList').addEventListener('click', async () => {
  try { setOut('loading...'); const s = await api('/api/codex/query', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({prefix: ''}) }); setOut(s) } catch (e) { setOut(e.message) }
})

document.getElementById('btnGet').addEventListener('click', async () => {
  try {
    const hash = document.getElementById('objHash').value.trim();
    if (!hash) { setOut('enter a hash'); return }
    setOut('loading...');
    const obj = await api(`/api/codex/object?hash=${encodeURIComponent(hash)}`);
    setOut(obj);
  } catch (e) { setOut(e.message) }
})
