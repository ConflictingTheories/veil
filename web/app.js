// Veil - Universal Content Management System
// Frontend Application Logic

let currentNode = null;
let allNodes = [];
let autoSaveTimer = null;
let autoSaveEnabled = true;
let autoSaveInterval = 5000;
let nodes_cache = {};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    setupEventListeners();
    loadNodes();
    setupAutoSave();
    restoreSettings();
});

// === Setup & Initialization ===
function setupEventListeners() {
    // Toolbar buttons
    document.getElementById('boldBtn').onclick = () => insertMarkdown('**', '**', 'bold text');
    document.getElementById('italicBtn').onclick = () => insertMarkdown('*', '*', 'italic text');
    document.getElementById('codeBtn').onclick = () => insertMarkdown('`', '`', 'code');
    document.getElementById('linkBtn').onclick = () => document.getElementById('linkModal').classList.remove('hidden');
    
    document.getElementById('publishBtn').onclick = () => document.getElementById('publishModal').classList.remove('hidden');
    document.getElementById('exportBtn').onclick = () => document.getElementById('exportModal').classList.remove('hidden');
    document.getElementById('versionBtn').onclick = showVersions;
    document.getElementById('tagsBtn').onclick = () => document.getElementById('tagsModal').classList.remove('hidden');
    document.getElementById('citationBtn').onclick = () => document.getElementById('citationModal').classList.remove('hidden');
    document.getElementById('mediaBtn').onclick = () => document.getElementById('mediaModal').classList.remove('hidden');
    
    document.getElementById('autoSaveToggle').onclick = toggleAutoSave;
    document.getElementById('settingsBtn').onclick = () => document.getElementById('settingsModal').classList.remove('hidden');
    
    // Sidebar
    document.getElementById('newNodeBtn').onclick = createNewNode;
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.onclick = (e) => switchSidebarTab(e.target.dataset.tab);
    });
    
    // Search
    document.getElementById('searchInput').addEventListener('input', (e) => filterNodes(e.target.value));
    
    // Editor
    document.getElementById('editor').addEventListener('input', debounce(() => {
        if (currentNode) {
            currentNode.title = document.getElementById('editor').value.split('\n')[0] || 'Untitled';
            updatePreview();
            if (autoSaveEnabled) triggerAutoSave();
        }
    }, 300));
    
    // Modal close buttons
    document.querySelectorAll('.modal-close').forEach(btn => {
        btn.onclick = (e) => e.target.closest('.modal').classList.add('hidden');
    });
    
    document.querySelectorAll('.modal .btn-secondary').forEach(btn => {
        btn.onclick = (e) => {
            let modal = e.target.closest('.modal');
            modal.classList.add('hidden');
        };
    });
    
    // Modal specific handlers
    document.getElementById('confirmPublish').onclick = publishNode;
    document.getElementById('confirmExport').onclick = exportNode;
    document.getElementById('addTagBtn').onclick = addTagToNode;
    document.getElementById('uploadMediaBtn').onclick = uploadMedia;
    document.getElementById('addCitationBtn').onclick = addCitation;
    
    // Settings
    document.getElementById('settingsBtn').onclick = openSettings;
    document.getElementById('autoSaveCheckbox').onchange = (e) => {
        autoSaveEnabled = e.target.checked;
        localStorage.setItem('autoSaveEnabled', autoSaveEnabled);
    };
    document.getElementById('autoSaveInterval').onchange = (e) => {
        autoSaveInterval = e.target.value * 1000;
        localStorage.setItem('autoSaveInterval', autoSaveInterval);
    };
    document.getElementById('darkModeToggle').onchange = (e) => {
        document.body.classList.toggle('dark-mode');
        localStorage.setItem('darkMode', e.target.checked);
    };
}

// === Node Management ===
async function loadNodes() {
    try {
        const resp = await fetch('/api/nodes');
        allNodes = await resp.json() || [];
        nodes_cache = {};
        allNodes.forEach(n => nodes_cache[n.id] = n);
        renderNodeTree();
    } catch (e) {
        console.error('Failed to load nodes:', e);
    }
}

function renderNodeTree() {
    const tree = document.getElementById('fileTree');
    tree.innerHTML = '';
    
    const rootNodes = allNodes.filter(n => !n.parent_id);
    rootNodes.forEach(node => {
        const item = document.createElement('div');
        item.className = 'tree-item';
        item.textContent = node.title || node.path || 'Untitled';
        item.onclick = () => openNode(node);
        tree.appendChild(item);
    });
}

function filterNodes(query) {
    const filtered = allNodes.filter(n => 
        n.title?.includes(query) || n.content?.includes(query) || n.path?.includes(query)
    );
    
    const tree = document.getElementById('fileTree');
    tree.innerHTML = '';
    filtered.forEach(node => {
        const item = document.createElement('div');
        item.className = 'tree-item';
        item.textContent = node.title || 'Untitled';
        item.onclick = () => openNode(node);
        tree.appendChild(item);
    });
}

async function openNode(node) {
    currentNode = node;
    document.getElementById('editor').value = node.content || '';
    document.title = `${node.title} - Veil`;
    updatePreview();
    loadReferences();
    loadVersions();
    loadTags();
    updateBreadcrumb(node);
    document.getElementById('statusText').textContent = `Opened: ${node.title}`;
}

function createNewNode() {
    const title = prompt('Node title:');
    if (!title) return;
    
    fetch('/api/node-create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            type: 'note',
            path: title.toLowerCase().replace(/\s+/g, '-') + '.md',
            title: title,
            content: '# ' + title + '\n\n',
            mime_type: 'text/markdown'
        })
    }).then(() => loadNodes()).catch(e => console.error(e));
}

async function saveCurrentNode() {
    if (!currentNode) return;
    
    currentNode.content = document.getElementById('editor').value;
    
    await fetch('/api/node-update', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(currentNode)
    });
    
    document.getElementById('statusText').textContent = '✓ Saved: ' + currentNode.title;
}

// === Auto-save ===
function setupAutoSave() {
    autoSaveCheckbox = document.getElementById('autoSaveCheckbox');
    autoSaveCheckbox.checked = localStorage.getItem('autoSaveEnabled') !== 'false';
    autoSaveEnabled = autoSaveCheckbox.checked;
    
    const interval = localStorage.getItem('autoSaveInterval');
    if (interval) autoSaveInterval = parseInt(interval);
}

function toggleAutoSave() {
    autoSaveEnabled = !autoSaveEnabled;
    const btn = document.getElementById('autoSaveToggle');
    btn.classList.toggle('active');
    localStorage.setItem('autoSaveEnabled', autoSaveEnabled);
    document.getElementById('statusText').textContent = 'Auto-save ' + (autoSaveEnabled ? 'enabled' : 'disabled');
}

function triggerAutoSave() {
    clearTimeout(autoSaveTimer);
    autoSaveTimer = setTimeout(() => {
        if (currentNode && autoSaveEnabled) {
            saveCurrentNode();
        }
    }, autoSaveInterval);
}

// === Publishing ===
async function publishNode() {
    if (!currentNode) return;
    
    const isPublic = document.getElementById('publishPublic').checked;
    const toWebsite = document.getElementById('publishWebsite').checked;
    const toRSS = document.getElementById('publishRSS').checked;
    
    await fetch('/api/publish?node_id=' + currentNode.id, { method: 'POST' });
    
    if (isPublic) {
        await fetch('/api/visibility?node_id=' + currentNode.id + '&visibility=public', { method: 'PUT' });
    }
    
    document.getElementById('publishModal').classList.add('hidden');
    document.getElementById('statusText').textContent = '✓ Published: ' + currentNode.title;
}

// === Export ===
async function exportNode() {
    if (!currentNode) return;
    
    const format = document.querySelector('input[name="exportFormat"]:checked')?.value || 'zip';
    const includeMedia = document.getElementById('exportIncludeMedia').checked;
    
    const url = `/api/export?node_id=${currentNode.id}&format=${format}&media=${includeMedia ? '1' : '0'}`;
    window.location.href = url;
    
    document.getElementById('exportModal').classList.add('hidden');
}

// === Versions ===
async function showVersions() {
    if (!currentNode) return;
    
    try {
        const resp = await fetch('/api/versions?node_id=' + currentNode.id);
        const versions = await resp.json() || [];
        
        const list = document.getElementById('versionsList');
        list.innerHTML = versions.map(v => `
            <div class="version-item">
                <strong>v${v.version_number}</strong> - ${v.status}
                <small>${new Date(v.created_at * 1000).toLocaleString()}</small>
                <button onclick="rollbackVersion('${v.id}')">Restore</button>
            </div>
        `).join('');
        
        document.getElementById('versionsModal').classList.remove('hidden');
    } catch (e) {
        console.error('Failed to load versions:', e);
    }
}

async function rollbackVersion(versionId) {
    await fetch('/api/rollback?version_id=' + versionId, { method: 'POST' });
    location.reload();
}

// === References & Linking ===
async function loadReferences() {
    if (!currentNode) return;
    
    try {
        // Load backlinks
        const backResp = await fetch('/api/backlinks/' + currentNode.id);
        const backlinks = await backResp.json() || [];
        
        // Load forward links
        const fwdResp = await fetch('/api/references?source=' + currentNode.id);
        const references = await fwdResp.json() || [];
        
        document.getElementById('linkedFromList').innerHTML = backlinks.map(r => 
            `<li><a onclick="openNode(nodes_cache['${r.source_node_id}'])">${nodes_cache[r.source_node_id]?.title}</a></li>`
        ).join('') || '<li>No backlinks</li>';
        
        document.getElementById('linksToList').innerHTML = references.map(r => 
            `<li><a onclick="openNode(nodes_cache['${r.target_node_id}'])">${nodes_cache[r.target_node_id]?.title}</a></li>`
        ).join('') || '<li>No links</li>';
    } catch (e) {
        console.error('Failed to load references:', e);
    }
}

function insertWikiLink(nodeId) {
    const editor = document.getElementById('editor');
    const node = nodes_cache[nodeId];
    if (!node) return;
    
    const link = `[[${node.title}]]`;
    editor.value += link;
    editor.focus();
    triggerAutoSave();
    document.getElementById('linkModal').classList.add('hidden');
}

// === Tags ===
async function loadTags() {
    if (!currentNode) return;
    
    try {
        const resp = await fetch('/api/node-tags?node_id=' + currentNode.id);
        const tags = await resp.json() || [];
        
        document.getElementById('nodeTagsList').innerHTML = tags.map(t => 
            `<span class="tag-chip" style="background:${t.color}">${t.name}</span>`
        ).join('');
    } catch (e) {
        console.error('Failed to load tags:', e);
    }
}

async function addTagToNode() {
    if (!currentNode) return;
    
    const name = document.getElementById('newTagInput').value;
    const color = document.getElementById('tagColorInput').value;
    
    if (!name) return;
    
    // Create tag and associate with node
    // Implementation would call API to create tag and add to node
    
    document.getElementById('newTagInput').value = '';
    loadTags();
}

// === Media ===
async function uploadMedia() {
    const files = document.getElementById('mediaUpload').files;
    
    for (let file of files) {
        const form = new FormData();
        form.append('file', file);
        
        try {
            const resp = await fetch('/api/media-upload', {
                method: 'POST',
                body: form
            });
            const media = await resp.json();
            console.log('Uploaded:', media);
        } catch (e) {
            console.error('Upload failed:', e);
        }
    }
    
    document.getElementById('mediaUpload').value = '';
    loadMediaLibrary();
}

async function loadMediaLibrary() {
    try {
        const resp = await fetch('/api/media-library?user_id=current');
        const media = await resp.json() || [];
        
        document.getElementById('mediaGrid').innerHTML = media.map(m => 
            `<div class="media-item"><strong>${m.filename}</strong></div>`
        ).join('');
    } catch (e) {
        console.error('Failed to load media:', e);
    }
}

// === Citations ===
async function addCitation() {
    const citation = {
        citation_key: document.getElementById('citationKey').value,
        authors: document.getElementById('citationAuthors').value,
        title: document.getElementById('citationTitle').value,
        year: parseInt(document.getElementById('citationYear').value) || 0,
        doi: document.getElementById('citationDOI').value,
        url: document.getElementById('citationURL').value,
        citation_format: document.getElementById('citationFormat').value
    };
    
    // Call API to save citation
    console.log('Citation:', citation);
    
    // Clear form
    Object.keys(citation).forEach(k => {
        const el = document.getElementById('citation' + k.charAt(0).toUpperCase() + k.slice(1));
        if (el) el.value = '';
    });
}

// === Preview & Rendering ===
function updatePreview() {
    const content = document.getElementById('editor').value;
    const html = markdownToHtml(content);
    document.getElementById('preview').innerHTML = html;
    updateWordCount(content);
}

function markdownToHtml(md) {
    let html = md;
    // Headers
    html = html.replace(/^### (.*?)$/gm, '<h3>$1</h3>');
    html = html.replace(/^## (.*?)$/gm, '<h2>$1</h2>');
    html = html.replace(/^# (.*?)$/gm, '<h1>$1</h1>');
    // Bold/Italic
    html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    html = html.replace(/__(.*?)__/g, '<strong>$1</strong>');
    html = html.replace(/\*(.*?)\*/g, '<em>$1</em>');
    html = html.replace(/_(.*?)_/g, '<em>$1</em>');
    // Code
    html = html.replace(/`(.*?)`/g, '<code>$1</code>');
    // Line breaks
    html = html.replace(/\n/g, '<br>');
    // Wiki links
    html = html.replace(/\[\[(.*?)\]\]/g, '<a href="#">[[$1]]</a>');
    return html;
}

function updateWordCount(text) {
    const words = text.trim().split(/\s+/).length;
    document.getElementById('wordCount').textContent = words;
}

function updateBreadcrumb(node) {
    document.getElementById('breadcrumb').textContent = node.path || node.title || 'Untitled';
}

// === Settings ===
function openSettings() {
    document.getElementById('settingsModal').classList.remove('hidden');
}

function restoreSettings() {
    const darkMode = localStorage.getItem('darkMode') === 'true';
    if (darkMode) {
        document.body.classList.add('dark-mode');
        document.getElementById('darkModeToggle').checked = true;
    }
    
    const interval = localStorage.getItem('autoSaveInterval');
    if (interval) {
        document.getElementById('autoSaveInterval').value = parseInt(interval) / 1000;
    }
}

// === Sidebar ===
function switchSidebarTab(tab) {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    
    event.target.classList.add('active');
    document.getElementById(tab + '-tab').classList.add('active');
}

// === Utilities ===
function debounce(fn, delay) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => fn(...args), delay);
    };
}

function insertMarkdown(before, after, placeholder) {
    const editor = document.getElementById('editor');
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const selected = editor.value.substring(start, end) || placeholder;
    
    const text = editor.value.substring(0, start) + before + selected + after + editor.value.substring(end);
    editor.value = text;
    editor.focus();
    editor.selectionStart = start + before.length;
    editor.selectionEnd = start + before.length + selected.length;
    triggerAutoSave();
}
