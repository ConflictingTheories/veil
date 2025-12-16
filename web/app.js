// Veil - Your Personal OS
// Frontend: Universal Content Platform with URI-First Architecture
// Every entity has a URI: veil://site/path/entity

let currentSite = null;
let currentNode = null;
let allNodes = [];
let allSites = [];
let autoSaveTimer = null;
let autoSaveEnabled = true;

// ====== INITIALIZATION ======
document.addEventListener('DOMContentLoaded', async () => {
    setupEventListeners();
    await loadSites();
    await loadNodes();
    restoreSettings();
    console.log('✓ Veil initialized');
});

// ====== EVENT SETUP ======
function setupEventListeners() {
    // Sidebar
    document.getElementById('newSiteBtn')?.addEventListener('click', createNewSite);
    document.getElementById('newNoteBtn')?.addEventListener('click', createNewNote);
    document.getElementById('settingsBtn')?.addEventListener('click', () => openModal('settingsModal'));
    document.getElementById('globalSearch')?.addEventListener('input', (e) => filterNodes(e.target.value));
    
    // Editor
    document.getElementById('editor')?.addEventListener('input', debounce(() => {
        if (currentNode) {
            currentNode.title = document.getElementById('editor').value.split('\n')[0] || 'Untitled';
            updatePreview();
            updateWordCount();
            if (autoSaveEnabled) triggerAutoSave();
        }
    }, 500));
    
    // Toolbar
    document.getElementById('boldBtn')?.addEventListener('click', () => insertMarkdown('**', '**', 'bold text'));
    document.getElementById('italicBtn')?.addEventListener('click', () => insertMarkdown('*', '*', 'italic text'));
    document.getElementById('codeBtn')?.addEventListener('click', () => insertMarkdown('`', '`', 'code'));
    document.getElementById('linkBtn')?.addEventListener('click', () => openModal('linkModal'));
    document.getElementById('tagsBtn')?.addEventListener('click', () => openModal('tagsModal'));
    document.getElementById('mediaBtn')?.addEventListener('click', () => openModal('mediaModal'));
    document.getElementById('versionsBtn')?.addEventListener('click', showVersions);
    document.getElementById('publishBtn')?.addEventListener('click', () => openModal('publishModal'));
    
    // Modals
    document.querySelectorAll('[data-modal-close]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const modal = e.target.closest('.modal-overlay');
            if (modal) closeModal(modal.id);
        });
    });
    
    document.querySelectorAll('.modal-overlay').forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) closeModal(modal.id);
        });
    });
    
    document.getElementById('confirmPublish')?.addEventListener('click', publishNode);
    document.getElementById('autoSaveToggle')?.addEventListener('change', (e) => {
        autoSaveEnabled = e.target.checked;
        localStorage.setItem('autoSaveEnabled', autoSaveEnabled);
    });
}

// ====== SITE MANAGEMENT ======
async function loadSites() {
    try {
        const resp = await fetch('/api/sites');
        allSites = await resp.json() || [];
        renderSitesList();
    } catch (e) {
        console.error('Failed to load sites:', e);
    }
}

function renderSitesList() {
    const list = document.getElementById('sitesList');
    if (!list) return;
    
    list.innerHTML = (allSites || []).map(site => `
        <div class="p-3 rounded-lg hover:bg-indigo-50 cursor-pointer transition border-l-4 ${
            site.id === currentSite?.id ? 'border-indigo-600 bg-indigo-50' : 'border-transparent'
        }" onclick="selectSite('${site.id}')">
            <div class="font-medium text-slate-900 text-sm">${site.name}</div>
            <div class="text-xs text-slate-500 mt-1">${site.description || 'No description'}</div>
        </div>
    `).join('');
}

async function selectSite(siteId) {
    const site = allSites.find(s => s.id === siteId);
    if (!site) return;
    
    currentSite = site;
    currentNode = null;
    document.getElementById('editor').value = '';
    document.getElementById('breadcrumb').innerHTML = `<span>${site.name}</span>`;
    document.title = `${site.name} - Veil`;
    
    renderSitesList();
    await loadNodes();
}

async function createNewSite() {
    const name = prompt('Site name (e.g., "My Portfolio", "Blog", "Game Dev"):');
    if (!name) return;
    
    const description = prompt('Brief description:') || '';
    
    try {
        const resp = await fetch('/api/sites', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, description, type: 'project' })
        });
        const site = await resp.json();
        allSites.push(site);
        renderSitesList();
        await selectSite(site.id);
        alert(`✓ Site created: ${name}`);
    } catch (e) {
        console.error('Failed to create site:', e);
        alert('Failed to create site');
    }
}

// ====== NODE MANAGEMENT ======
async function loadNodes() {
    if (!currentSite) return;
    
    try {
        const resp = await fetch(`/api/sites/${currentSite.id}/nodes`);
        allNodes = await resp.json() || [];
        renderNodesList();
    } catch (e) {
        console.error('Failed to load nodes:', e);
    }
}

function renderNodesList() {
    const list = document.getElementById('nodesList');
    if (!list) return;
    
    list.innerHTML = (allNodes || []).map(n => `
        <div onclick="openNode('${n.id}')" 
            class="p-3 rounded-lg hover:bg-indigo-50 cursor-pointer transition border-l-4 ${
                n.id === currentNode?.id ? 'border-indigo-600 bg-indigo-50' : 'border-transparent'
            }">
            <div class="font-medium text-slate-900 text-sm flex items-center gap-2">
                <span class="text-xs px-2 py-0.5 bg-slate-200 rounded-full">${n.type || 'note'}</span>
                ${n.title || 'Untitled'}
            </div>
            <div class="text-xs text-slate-500 mt-1">${n.path}</div>
        </div>
    `).join('');
}

function filterNodes(query) {
    if (!query) {
        renderNodesList();
        return;
    }
    
    const filtered = (allNodes || []).filter(n =>
        n.title?.toLowerCase().includes(query.toLowerCase()) ||
        n.content?.toLowerCase().includes(query.toLowerCase())
    );
    
    const list = document.getElementById('nodesList');
    list.innerHTML = filtered.map(n => `
        <div onclick="openNode('${n.id}')" class="p-3 rounded-lg hover:bg-indigo-50 cursor-pointer transition border-l-4 border-transparent">
            <div class="font-medium text-slate-900 text-sm">${n.title || 'Untitled'}</div>
            <div class="text-xs text-slate-500 mt-1">${n.path}</div>
        </div>
    `).join('');
}

async function openNode(nodeId) {
    try {
        const resp = await fetch(`/api/sites/${currentSite.id}/nodes/${nodeId}`);
        currentNode = await resp.json();
        
        document.getElementById('editor').value = currentNode.content || '';
        document.getElementById('breadcrumb').innerHTML = `<span>${currentNode.title || 'Untitled'}</span>`;
        document.title = `${currentNode.title} - ${currentSite.name} - Veil`;
        
        updatePreview();
        updateWordCount();
        loadReferences();
        
        renderNodesList();
    } catch (e) {
        console.error('Failed to open node:', e);
    }
}

async function createNewNote() {
    if (!currentSite) {
        alert('Please select or create a site first');
        return;
    }
    
    const title = prompt('Note title:');
    if (!title) return;
    
    try {
        const resp = await fetch(`/api/sites/${currentSite.id}/nodes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                type: 'note',
                path: title.toLowerCase().replace(/\s+/g, '-') + '.md',
                title: title,
                content: '# ' + title + '\n\n',
                mime_type: 'text/markdown'
            })
        });
        const node = await resp.json();
        allNodes.push(node);
        renderNodesList();
        await openNode(node.id);
        alert(`✓ Note created: ${title}`);
    } catch (e) {
        console.error('Failed to create note:', e);
        alert('Failed to create note');
    }
}

// ====== EDITING ======
async function saveCurrentNode() {
    if (!currentNode || !currentSite) return;
    
    currentNode.content = document.getElementById('editor').value;
    currentNode.title = currentNode.content.split('\n')[0] || 'Untitled';
    
    try {
        await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(currentNode)
        });
        
        showStatusBadge('Saved', 'green');
    } catch (e) {
        console.error('Save failed:', e);
        showStatusBadge('Save failed', 'red');
    }
}

function triggerAutoSave() {
    clearTimeout(autoSaveTimer);
    autoSaveTimer = setTimeout(() => {
        if (currentNode && autoSaveEnabled) saveCurrentNode();
    }, 3000);
}

// ====== PREVIEW ======
function updatePreview() {
    const content = document.getElementById('editor').value;
    const html = markdownToHtml(content);
    document.getElementById('preview').innerHTML = html;
}

function markdownToHtml(md) {
    let html = md
        .replace(/^### (.*?)$/gm, '<h3 class="text-lg font-bold mt-4 mb-2">$1</h3>')
        .replace(/^## (.*?)$/gm, '<h2 class="text-xl font-bold mt-4 mb-2">$1</h2>')
        .replace(/^# (.*?)$/gm, '<h1 class="text-2xl font-bold mt-4 mb-2">$1</h1>')
        .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
        .replace(/__(.*?)__/g, '<strong>$1</strong>')
        .replace(/\*(.*?)\*/g, '<em>$1</em>')
        .replace(/_(.*?)_/g, '<em>$1</em>')
        .replace(/`([^`]*)`/g, '<code class="bg-slate-100 px-2 py-1 rounded text-red-600 text-sm">$1</code>')
        .replace(/\[\[(.*?)\]\]/g, '<a href="#" class="text-indigo-600 hover:underline">$1</a>')
        .replace(/\n\n/g, '</p><p class="mb-3">')
        .replace(/\n/g, '<br>');
    return '<p class="mb-3 text-slate-700">' + html + '</p>';
}

function updateWordCount() {
    const text = document.getElementById('editor').value;
    const words = text.trim().split(/\s+/).filter(w => w.length > 0).length;
    const chars = text.length;
    
    const countEl = document.getElementById('wordCount');
    if (countEl) countEl.textContent = `${words} words • ${chars} characters`;
}

// ====== VERSIONING ======
async function showVersions() {
    if (!currentNode || !currentSite) return;
    
    try {
        const resp = await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/versions`);
        const versions = await resp.json() || [];
        
        const list = document.getElementById('versionsList');
        list.innerHTML = (versions || []).map(v => `
            <div class="p-3 bg-slate-50 rounded-lg border border-slate-200">
                <div class="flex justify-between items-start">
                    <div>
                        <div class="font-medium text-sm">v${v.version_number}</div>
                        <div class="text-xs text-slate-500">${new Date(v.created_at * 1000).toLocaleString()}</div>
                        <div class="text-xs text-slate-600 mt-1 capitalize">${v.status}</div>
                    </div>
                    <button onclick="rollbackVersion('${v.id}')" 
                        class="px-2 py-1 bg-indigo-600 text-white text-xs rounded hover:bg-indigo-700">
                        Restore
                    </button>
                </div>
            </div>
        `).join('');
        
        openModal('versionsModal');
    } catch (e) {
        console.error('Failed to load versions:', e);
    }
}

async function rollbackVersion(versionId) {
    if (!currentNode || !currentSite) return;
    
    try {
        await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/versions/${versionId}/rollback`, {
            method: 'POST'
        });
        await openNode(currentNode.id);
        alert('✓ Restored to previous version');
    } catch (e) {
        console.error('Rollback failed:', e);
        alert('Failed to rollback');
    }
}

// ====== REFERENCES ======
async function loadReferences() {
    if (!currentNode || !currentSite) return;
    
    try {
        const backResp = await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/backlinks`);
        const backlinks = await backResp.json() || [];
        
        const fwdResp = await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/references`);
        const references = await fwdResp.json() || [];
        
        document.getElementById('backlinks').innerHTML = (backlinks || []).length ?
            (backlinks || []).map(r => `<li class="text-indigo-600 cursor-pointer hover:underline">← ${r.link_text || 'Link'}</li>`).join('') :
            '<li class="text-slate-500">No backlinks</li>';
        
        document.getElementById('forwardlinks').innerHTML = (references || []).length ?
            (references || []).map(r => `<li class="text-indigo-600 cursor-pointer hover:underline">→ ${r.link_text || 'Link'}</li>`).join('') :
            '<li class="text-slate-500">No links</li>';
    } catch (e) {
        console.error('Failed to load references:', e);
    }
}

// ====== PUBLISHING ======
async function publishNode() {
    if (!currentNode || !currentSite) return;
    
    const isPublic = document.getElementById('publishPublic')?.checked || false;
    
    try {
        await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/publish`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ visibility: isPublic ? 'public' : 'private' })
        });
        
        closeModal('publishModal');
        alert('✓ Published: ' + currentNode.title);
        showStatusBadge('Published', 'green');
    } catch (e) {
        console.error('Publish failed:', e);
        alert('Failed to publish');
    }
}

// ====== TAGS ======
async function addTag() {
    if (!currentNode || !currentSite) return;
    
    const name = document.getElementById('newTagInput')?.value;
    if (!name) return;
    
    try {
        await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/tags`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        
        document.getElementById('newTagInput').value = '';
        alert(`✓ Tag added: ${name}`);
    } catch (e) {
        console.error('Failed to add tag:', e);
    }
}

// ====== UI UTILITIES ======
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
    updatePreview();
}

function openModal(id) {
    const modal = document.getElementById(id);
    if (modal) modal.classList.remove('hidden');
}

function closeModal(id) {
    const modal = document.getElementById(id);
    if (modal) modal.classList.add('hidden');
}

function showStatusBadge(message, color = 'green') {
    const badge = document.getElementById('statusBadge');
    if (!badge) return;
    
    const colorMap = {
        green: 'bg-green-100 text-green-700',
        red: 'bg-red-100 text-red-700',
        yellow: 'bg-yellow-100 text-yellow-700'
    };
    
    badge.className = `text-xs px-3 py-1 rounded-full ${colorMap[color]}`;
    badge.innerHTML = `<i class="fas fa-circle animate-pulse mr-1"></i>${message}`;
}

function restoreSettings() {
    const autoSave = localStorage.getItem('autoSaveEnabled') !== 'false';
    const toggle = document.getElementById('autoSaveToggle');
    if (toggle) {
        toggle.checked = autoSave;
        autoSaveEnabled = autoSave;
    }
}

function debounce(fn, delay) {
    let timeout;
    return function (...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => fn(...args), delay);
    };
}
