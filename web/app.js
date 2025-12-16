let nodes = [], currentNode = null, previewVisible = false;
async function loadNodes() {
    try {
        const r = await fetch('/api/nodes'); nodes = await r.json();
        renderTree(); if (nodes.length > 0) openNode(nodes[0])
    } catch (e) { console.error(e) }
}
function renderTree() {
    const t = document.getElementById('treeView'); t.innerHTML = '';
    nodes.forEach(n => {
        const i = document.createElement('div'); i.className = 'tree-item';
        i.innerHTML = `<span>${n.type === 'note' ? '📄' : '📁'}</span><span>${n.path}</span>`;
        i.onclick = () => openNode(n); t.appendChild(i)
    })
}
async function openNode(n) {
    try {
        const r = await fetch(`/api/node/${n.id}`);
        currentNode = await r.json(); document.getElementById('editor').value = currentNode.content || '';
        updatePreview(); document.querySelectorAll('.tree-item').forEach((i, idx) =>
            i.classList.toggle('active', idx === nodes.indexOf(n)));
        document.getElementById('tabs').innerHTML = `<div class="tab active"><span>${currentNode.path}</span></div>`
    }
    catch (e) { console.error(e) }
}
async function saveNode() {
    if (!currentNode) return; currentNode.content = document.getElementById('editor').value;
    try {
        await fetch(`/api/node/${currentNode.id}`, {
            method: 'PUT', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(currentNode)
        }); document.getElementById('syncText').textContent = 'Saved';
        setTimeout(() => document.getElementById('syncText').textContent = 'Ready', 2000)
    } catch (e) { console.error(e) }
}
function updatePreview() {
    const c = document.getElementById('editor').value;
    const p = document.getElementById('preview'); p.innerHTML = c.replace(/^# (.*$)/gim, '<h1>$1</h1>')
        .replace(/^## (.*$)/gim, '<h2>$1</h2>').replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
        .replace(/\n/g, '<br>')
}
let saveTimeout; document.getElementById('editor').addEventListener('input', () => {
    updatePreview(); clearTimeout(saveTimeout); saveTimeout = setTimeout(saveNode, 1000)
});
document.getElementById('togglePreview').addEventListener('click', () => {
    previewVisible = !previewVisible; document.getElementById('preview').classList.toggle('active', previewVisible)
});
document.getElementById('newFileBtn').addEventListener('click', async () => {
    const p = prompt('Enter file path:'); if (!p) return; try {
        const r = await fetch('/api/nodes',
            {
                method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({
                    type: 'note',
                    path: p, title: p.split('/').pop(), content: '', mime_type: 'text/markdown'
                })
            });
        const n = await r.json(); nodes.push(n); renderTree(); openNode(n)
    } catch (e) { console.error(e) }
});
document.addEventListener('keydown', e => { if ((e.ctrlKey || e.metaKey) && e.key === 's') { e.preventDefault(); saveNode() } });
loadNodes();
