// Veil - Your Personal OS
// Frontend: Universal Content Platform with URI-First Architecture
// Every entity has a URI: veil://site/path/entity

let currentSite = null;
let currentNode = null;
let allNodes = [];
let allSites = [];
let autoSaveTimer = null;
let autoSaveEnabled = true;

// Node types for Veil
const NODE_TYPES = {
    note: 'note',
    page: 'page',
    post: 'post',
    canvas: 'canvas',
    'shader-demo': 'shader-demo',
    'code-snippet': 'code-snippet',
    image: 'image',
    video: 'video',
    audio: 'audio',
    document: 'document',
    todo: 'todo',
    reminder: 'reminder'
};

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
    document.getElementById('pluginsBtn')?.addEventListener('click', async () => { openModal('pluginsModal'); await loadPlugins(); });
    document.getElementById('exportBtn')?.addEventListener('click', () => openModal('exportModal'));
    document.getElementById('openSiteEditor')?.addEventListener('click', () => { openModal('siteEditorModal'); populateSiteEditor(); });
    document.getElementById('saveSiteBtn')?.addEventListener('click', saveSiteEdits);
    document.getElementById('addUriBtn')?.addEventListener('click', addUriPrompt);
    document.getElementById('globalSearch')?.addEventListener('input', (e) => filterNodes(e.target.value));
    document.getElementById('exportSiteBtn')?.addEventListener('click', exportCurrentSite);
    
    // Editor
    document.getElementById('editor')?.addEventListener('input', debounce(() => {
        if (currentNode) {
            currentNode.title = document.getElementById('editor').value.split('\n')[0] || 'Untitled';
            updatePreview();
            updateWordCount();
            if (autoSaveEnabled) {
                console.debug('Editor changed: scheduling autosave');
                triggerAutoSave();
            }
        }
    }, 500));

    // Save button (manual)
    document.getElementById('saveBtn')?.addEventListener('click', async () => {
        showStatusBadge('Saving...', 'yellow');
        await saveCurrentNode();
    });

    // Toolbar
    document.getElementById('boldBtn')?.addEventListener('click', () => insertMarkdown('**', '**', 'bold text'));
    document.getElementById('italicBtn')?.addEventListener('click', () => insertMarkdown('*', '*', 'italic text'));
    document.getElementById('codeBtn')?.addEventListener('click', () => insertMarkdown('`', '`', 'code'));
    document.getElementById('linkBtn')?.addEventListener('click', () => openModal('linkModal'));
    document.getElementById('tagsBtn')?.addEventListener('click', () => openModal('tagsModal'));
    document.getElementById('mediaBtn')?.addEventListener('click', () => openModal('mediaModal'));
    document.getElementById('versionsBtn')?.addEventListener('click', showVersions);
    document.getElementById('publishBtn')?.addEventListener('click', () => openModal('publishModal'));
    document.getElementById('svgBtn')?.addEventListener('click', () => createSVGCanvas());
    
    // Modals
    document.querySelectorAll('[data-modal-close]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            // Use the button element (btn) to find the containing overlay reliably
            const modal = btn.closest('.modal-overlay');
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
        if (!resp.ok) {
            const txt = await resp.text().catch(() => resp.statusText || 'Unknown error');
            console.error('Failed to load sites:', resp.status, txt);
            allSites = [];
            renderSitesList();
            return;
        }

        const data = await resp.json();
        // Accept either an array or an object like { sites: [...] }
        if (Array.isArray(data)) {
            allSites = data;
        } else if (data && Array.isArray(data.sites)) {
            allSites = data.sites;
        } else {
            // Unexpected shape - reset to empty array and log
            console.warn('Unexpected /api/sites response shape, resetting sites list', data);
            allSites = [];
        }

        renderSitesList();
    } catch (e) {
        console.error('Failed to load sites:', e);
        allSites = [];
        renderSitesList();
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
    const site = (allSites || []).find(s => s.id === siteId);
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

        if (!resp.ok) {
            const errBody = await resp.json().catch(() => ({ error: resp.statusText }));
            console.error('Create site failed:', resp.status, errBody);
            alert('Failed to create site: ' + (errBody.error || resp.statusText));
            return;
        }

        const site = await resp.json();
        // Ensure allSites is an array before pushing
        if (!Array.isArray(allSites)) allSites = [];
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
        if (!resp.ok) {
            const txt = await resp.text().catch(() => resp.statusText || 'Unknown error');
            console.error('Failed to load nodes:', resp.status, txt);
            allNodes = [];
            renderNodesList();
            return;
        }

        const data = await resp.json();
        if (Array.isArray(data)) {
            allNodes = data;
        } else if (data && Array.isArray(data.nodes)) {
            allNodes = data.nodes;
        } else {
            console.warn('Unexpected /api/sites/:id/nodes response shape, resetting nodes list', data);
            allNodes = [];
        }

        renderNodesList();
    } catch (e) {
        console.error('Failed to load nodes:', e);
        allNodes = [];
        renderNodesList();
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
        if (!resp.ok) {
            const body = await resp.text().catch(() => resp.statusText || 'Unknown error');
            console.error('Failed to open node, server responded', resp.status, body);
            alert('Failed to open node');
            return;
        }

        const node = await resp.json();
        currentNode = node;
        
        document.getElementById('editor').value = currentNode.content || '';
        document.getElementById('breadcrumb').innerHTML = `<span>${currentNode.title || 'Untitled'}</span>`;
        document.title = `${currentNode.title} - ${currentSite.name} - Veil`;
        
        updatePreview();
        updateWordCount();
        loadReferences();
        loadNodeURIs();
        
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

    async function tryCreate(noteTitle, attempt = 1) {
        try {
            const path = noteTitle.toLowerCase().replace(/\s+/g, '-') + '.md';
            const resp = await fetch(`/api/sites/${currentSite.id}/nodes`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    type: 'note',
                    path: path,
                    title: noteTitle,
                    content: '# ' + noteTitle + '\n\n',
                    mime_type: 'text/markdown'
                })
            });

            if (!resp.ok) {
                const errBody = await resp.json().catch(() => ({ error: resp.statusText }));
                console.error('Create note failed:', resp.status, errBody);

                // If it's a conflict (duplicate path), try once with a unique suffix
                if (resp.status === 409 && attempt === 1) {
                    const uniqueTitle = noteTitle + ' ' + Date.now();
                    return await tryCreate(uniqueTitle, 2);
                }

                alert('Failed to create note: ' + (errBody.error || resp.statusText));
                return;
            }

            const node = await resp.json();
            if (!Array.isArray(allNodes)) allNodes = [];
            // Reload nodes from server to ensure authoritative data (and site assignment)
            await loadNodes();
            // Find the created node in the refreshed list (prefer server-provided id)
            let created = null;
            if (node && node.id) {
                created = (allNodes || []).find(n => n.id === node.id);
            }
            if (!created) {
                // try match by path and title (best-effort)
                created = (allNodes || []).find(n => n.path === node.path && n.title === node.title);
            }
            if (created && created.id) {
                renderNodesList();
                await openNode(created.id);
                alert(`✓ Note created: ${created.title}`);
            } else if (node && node.id) {
                // Fallback to returned id if was provided
                renderNodesList();
                await openNode(node.id).catch(() => alert('Note created but failed to open'));
            } else {
                renderNodesList();
                alert('Note created but could not locate it in the site');
            }
        } catch (e) {
            console.error('Failed to create note:', e);
            alert('Failed to create note');
        }
    }

    await tryCreate(title);
}

// ====== EDITING ======
async function saveCurrentNode() {
    if (!currentNode || !currentSite) {
        console.warn('No current node/site to save');
        return;
    }
    
    currentNode.content = document.getElementById('editor').value;
    currentNode.title = currentNode.content.split('\n')[0] || 'Untitled';

    try {
        const resp = await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(currentNode)
        });

        if (!resp.ok) {
            const body = await resp.text().catch(() => resp.statusText || 'Unknown error');
            console.error('Save failed, server responded:', resp.status, body);
            showStatusBadge('Save failed', 'red');
            return;
        }

        // Update UI state
        showStatusBadge('Saved', 'green');
        // Update nodes list (title may have changed)
        // Persist local currentNode changes to allNodes array so UI reflects edits immediately
        if (Array.isArray(allNodes)) {
            const idx = allNodes.findIndex(n => n.id === currentNode.id);
            if (idx !== -1) {
                allNodes[idx].title = currentNode.title;
                allNodes[idx].content = currentNode.content;
            }
        }
        renderNodesList();
    } catch (e) {
        console.error('Save failed:', e);
        showStatusBadge('Save failed', 'red');
    }
}

function triggerAutoSave() {
    clearTimeout(autoSaveTimer);
    autoSaveTimer = setTimeout(() => {
        if (currentNode && autoSaveEnabled) {
            console.debug('Auto-save triggered');
            showStatusBadge('Auto-saving...', 'yellow');
            saveCurrentNode();
        }
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
    if (!currentNode || !currentSite || !currentNode.id) return;
    
    try {
        const backResp = await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/backlinks`);
        if (!backResp.ok) {
            document.getElementById('backlinks').innerHTML = '<li class="text-slate-500">Failed to load backlinks</li>';
        } else {
            const backlinks = await backResp.json() || [];
            document.getElementById('backlinks').innerHTML = (backlinks || []).length ?
                (backlinks || []).map(r => `<li class="text-indigo-600 cursor-pointer hover:underline">← ${r.link_text || 'Link'}</li>`).join('') :
                '<li class="text-slate-500">No backlinks</li>';
        }
        
        const fwdResp = await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/references`);
        if (!fwdResp.ok) {
            document.getElementById('forwardlinks').innerHTML = '<li class="text-slate-500">Failed to load links</li>';
        } else {
            const references = await fwdResp.json() || [];
            document.getElementById('forwardlinks').innerHTML = (references || []).length ?
                (references || []).map(r => `<li class="text-indigo-600 cursor-pointer hover:underline">→ ${r.link_text || 'Link'}</li>`).join('') :
                '<li class="text-slate-500">No links</li>';
        }
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

// ====== PLUGIN MANAGEMENT ======
async function loadPlugins() {
    try {
        const resp = await fetch('/api/plugins-registry');
        if (!resp.ok) {
            console.error('Failed to load plugins: ', resp.status);
            return;
        }
        const data = await resp.json();
        const list = document.getElementById('pluginsList');
        list.innerHTML = '';
        (data || []).forEach(p => {
            const el = document.createElement('div');
            el.className = 'flex items-center justify-between p-3 border rounded-lg';
            el.innerHTML = `<div class="text-sm"><div class="font-medium">${p.name}</div><div class="text-xs text-slate-500">${p.slug}</div></div><div><button class="toggle-btn px-3 py-1 rounded-lg text-sm">${p.enabled ? 'Disable' : 'Enable'}</button></div>`;
            el.querySelector('.toggle-btn').addEventListener('click', () => togglePlugin(p));
            list.appendChild(el);
        });
    } catch (e) {
        console.error('Failed to load plugins', e);
    }
}

async function togglePlugin(plugin) {
    try {
        // Flip enabled flag
        const updated = Object.assign({}, plugin, { enabled: !plugin.enabled });
        const resp = await fetch('/api/plugins-registry', {
            method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(updated)
        });
        if (!resp.ok) {
            const body = await resp.text().catch(() => resp.statusText);
            alert('Failed to update plugin: ' + body);
            return;
        }
        await loadPlugins();
    } catch (e) {
        console.error('togglePlugin failed', e);
    }
}

// ====== SITE EDITOR ======
function populateSiteEditor() {
    if (!currentSite) return;
    document.getElementById('siteNameInput').value = currentSite.name || '';
    document.getElementById('siteDescriptionInput').value = currentSite.description || '';
    document.getElementById('siteTypeInput').value = currentSite.type || 'project';
}

async function saveSiteEdits() {
    if (!currentSite) return;
    const name = document.getElementById('siteNameInput').value;
    const description = document.getElementById('siteDescriptionInput').value;
    const type = document.getElementById('siteTypeInput').value;

    try {
        const resp = await fetch('/api/sites', {
            method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: currentSite.id, name, description, type })
        });
        if (!resp.ok) {
            const body = await resp.text().catch(() => resp.statusText);
            alert('Failed to update site: ' + body);
            return;
        }
        const updated = await resp.json();
        // Refresh sites list
        await loadSites();
        // Update currentSite reference
        currentSite = updated;
        closeModal('siteEditorModal');
        alert('✓ Site updated');
    } catch (e) {
        console.error('Failed to save site edits', e);
        alert('Failed to update site');
    }
}

// ====== NODE URI MANAGEMENT ======
async function loadNodeURIs() {
    if (!currentNode) return;
    try {
        const resp = await fetch(`/api/node-uris?node_id=${encodeURIComponent(currentNode.id)}`);
        if (!resp.ok) {
            console.error('Failed to load URIs', resp.status);
            return;
        }
        const uris = await resp.json();
        const el = document.getElementById('nodeUris');
        el.innerHTML = '';
        if (!uris || uris.length === 0) {
            el.innerHTML = '<li class="text-slate-500">No URIs</li>';
            return;
        }
        uris.forEach(u => {
            const item = document.createElement('li');
            item.className = 'flex items-center justify-between';
            item.innerHTML = `<div class="text-sm"><div class="font-medium">${u.uri}</div></div><div><button class="text-xs text-red-600">Delete</button></div>`;
            item.querySelector('button').addEventListener('click', () => deleteNodeURI(u.id));
            el.appendChild(item);
        });
    } catch (e) {
        console.error('loadNodeURIs failed', e);
    }
}

function addUriPrompt() {
    if (!currentNode) { alert('Open a node first'); return; }
    const uri = prompt('Enter URI (e.g., /projects/my-portfolio)');
    if (!uri) return;
    addNodeURI(uri);
}

async function addNodeURI(uri) {
    try {
        const resp = await fetch('/api/node-uris', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ node_id: currentNode.id, uri }) });
        if (!resp.ok) { alert('Failed to add URI'); return; }
        await loadNodeURIs();
    } catch (e) { console.error('addNodeURI failed', e); }
}

async function deleteNodeURI(id) {
    try {
        const resp = await fetch(`/api/node-uris?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
        if (!resp.ok) { alert('Failed to delete URI'); return; }
        await loadNodeURIs();
    } catch (e) { console.error('deleteNodeURI failed', e); }
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
    if (modal) {
        modal.classList.remove('hidden');
        // Prevent background scroll while modal is open
        document.body.style.overflow = 'hidden';
        // Focus first interactive element for accessibility
        const first = modal.querySelector('input, button, textarea, [tabindex]');
        if (first) first.focus();
    }
}

function closeModal(id) {
    const modal = document.getElementById(id);
    if (modal) {
        modal.classList.add('hidden');
        // Restore scroll
        document.body.style.overflow = '';
    }
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

// ====== SVG CANVAS FUNCTIONS ======

async function createSVGCanvas() {
    const width = 800;
    const height = 600;
    const name = prompt('Enter SVG canvas name:', 'New Drawing');

    if (!name) return;

    try {
        const response = await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'svg',
                action: 'create',
                payload: { width, height, name }
            })
        });

        const result = await response.json();

        if (result.svg) {
            // Create a new node with the SVG content
            const nodeData = {
                type: 'canvas',
                title: name,
                content: result.svg,
                path: `drawings/${name.toLowerCase().replace(/\s+/g, '-')}`,
                mime_type: 'image/svg+xml'
            };

            const createResponse = await fetch(`/api/sites/${currentSite.id}/nodes`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(nodeData)
            });

            if (createResponse.ok) {
                showToast('SVG canvas created successfully!', 'success');
                await loadNodes(); // Refresh the node list
            } else {
                showToast('Failed to save SVG canvas', 'error');
            }
        }
    } catch (error) {
        console.error('SVG creation failed:', error);
        showToast('Failed to create SVG canvas', 'error');
    }
}

// ====== UTILITY FUNCTIONS ======

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
}

// ====== EXPORT FUNCTIONALITY ======

async function exportCurrentSite() {
    if (!currentSite) {
        alert('Please select a site first');
        return;
    }
    
    try {
        showStatusBadge('Exporting...', 'yellow');
        const url = `/api/export?site_id=${encodeURIComponent(currentSite.id)}&format=zip`;
        
        // Create a temporary link and trigger download
        const a = document.createElement('a');
        a.href = url;
        a.download = `veil-site-${currentSite.id}.zip`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        
        showStatusBadge('Exported!', 'green');
        closeModal('exportModal');
        showToast('Site exported successfully!', 'success');
    } catch (e) {
        console.error('Export failed:', e);
        showStatusBadge('Export failed', 'red');
        alert('Failed to export site');
    }
}

// ====== PLUGIN CONFIGURATION ======

async function saveGitConfig() {
    const repo = document.getElementById('gitRepo')?.value;
    const branch = document.getElementById('gitBranch')?.value;
    const token = document.getElementById('gitToken')?.value;
    
    if (!repo || !branch) {
        alert('Please fill in all required fields');
        return;
    }
    
    try {
        const response = await fetch('/api/plugins/git/configure', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                enabled: true,
                config: { repo, branch, token }
            })
        });
        
        if (response.ok) {
            alert('Git configuration saved');
        } else {
            alert('Failed to save configuration');
        }
    } catch (e) {
        console.error('Git config error:', e);
        alert('Configuration error');
    }
}

// ====== MEDIA FUNCTIONALITY ======

async function uploadMedia() {
    const fileInput = document.getElementById('mediaFileInput');
    const file = fileInput.files[0];
    if (!file) {
        alert('Please select a file');
        return;
    }
    
    const formData = new FormData();
    formData.append('file', file);
    
    try {
        const response = await fetch('/api/media/upload', {
            method: 'POST',
            body: formData
        });
        const data = await response.json();
        if (response.ok) {
            showToast('Media uploaded successfully', 'success');
            loadMediaLibrary();
            fileInput.value = '';
        } else {
            alert('Upload failed: ' + (data.error || 'Unknown error'));
        }
    } catch (err) {
        alert('Upload error: ' + err.message);
    }
}

async function loadMediaLibrary() {
    try {
        const response = await fetch('/api/media');
        const media = await response.json();
        const library = document.getElementById('mediaLibrary');
        
        if (!media || media.length === 0) {
            library.innerHTML = '<p class="col-span-3 text-center text-slate-500 text-sm py-8">No media uploaded yet</p>';
            return;
        }
        
        library.innerHTML = media.map(m => {
            const isImage = m.mime_type && m.mime_type.startsWith('image/');
            return `
                <div class="border border-slate-200 rounded-lg p-2 hover:border-indigo-500 cursor-pointer transition" onclick="insertMediaIntoEditor('${m.url}', '${m.filename}')">
                    ${isImage ? `<img src="${m.url}" alt="${m.filename}" class="w-full h-24 object-cover rounded mb-1">` : `<div class="w-full h-24 bg-slate-100 rounded mb-1 flex items-center justify-center"><i class="fas fa-file text-3xl text-slate-400"></i></div>`}
                    <p class="text-xs text-slate-600 truncate" title="${m.filename}">${m.filename}</p>
                    <p class="text-xs text-slate-400">${formatFileSize(m.size)}</p>
                </div>
            `;
        }).join('');
    } catch (err) {
        console.error('Failed to load media:', err);
    }
}

function formatFileSize(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

function insertMediaIntoEditor(url, filename) {
    const editor = document.getElementById('noteContent');
    if (!editor) return;
    
    const isImage = url.match(/\.(jpg|jpeg|png|gif|svg|webp)$/i);
    const markdown = isImage ? `![${filename}](${url})` : `[${filename}](${url})`;
    
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const text = editor.value;
    editor.value = text.substring(0, start) + markdown + text.substring(end);
    editor.selectionStart = editor.selectionEnd = start + markdown.length;
    editor.focus();
    closeModal('mediaModal');
}

// ====== LINK FUNCTIONALITY ======

let selectedLinkNode = null;

document.addEventListener('DOMContentLoaded', function() {
    const linkTypeSelect = document.getElementById('linkType');
    if (linkTypeSelect) {
        linkTypeSelect.addEventListener('change', (e) => {
            const type = e.target.value;
            document.getElementById('internalLinkSection').classList.toggle('hidden', type !== 'internal');
            document.getElementById('externalLinkSection').classList.toggle('hidden', type !== 'external');
            document.getElementById('veilUriSection').classList.toggle('hidden', type !== 'veil');
        });
    }
    
    const linkSearchInput = document.getElementById('linkSearchInput');
    if (linkSearchInput) {
        linkSearchInput.addEventListener('input', async (e) => {
            const query = e.target.value.toLowerCase();
            if (!query || !currentSite) return;
            
            try {
                const response = await fetch(`/api/sites/${currentSite}/nodes`);
                const data = await response.json();
                const nodes = data.nodes || [];
                const filtered = nodes.filter(n => 
                    n.title.toLowerCase().includes(query) || 
                    (n.content && n.content.toLowerCase().includes(query))
                );
                
                const results = document.getElementById('linkSearchResults');
                results.innerHTML = filtered.slice(0, 10).map(n => `
                    <div class="p-2 hover:bg-slate-100 rounded cursor-pointer text-sm" onclick="selectLinkNode('${n.id}', '${n.title.replace(/'/g, "\\'")}')">
                        <div class="font-medium text-slate-900">${n.title}</div>
                        <div class="text-xs text-slate-500">${n.type || 'note'}</div>
                    </div>
                `).join('');
            } catch (err) {
                console.error('Search failed:', err);
            }
        });
    }
});

function selectLinkNode(nodeId, title) {
    selectedLinkNode = { id: nodeId, title: title };
    document.getElementById('linkTextInput').value = title;
    document.querySelectorAll('#linkSearchResults > div').forEach(el => {
        el.classList.remove('bg-indigo-50', 'border-l-4', 'border-indigo-500');
    });
    event.target.closest('div').classList.add('bg-indigo-50', 'border-l-4', 'border-indigo-500');
}

async function insertLink() {
    const type = document.getElementById('linkType').value;
    const linkText = document.getElementById('linkTextInput').value;
    const editor = document.getElementById('noteContent');
    
    if (!editor || !linkText) {
        alert('Please provide link text');
        return;
    }
    
    let linkUrl = '';
    
    if (type === 'internal' && selectedLinkNode) {
        linkUrl = `veil://note/${selectedLinkNode.id}`;
        
        // Create reference in database
        if (currentNode) {
            try {
                await fetch(`/api/sites/${currentSite}/nodes/${currentNode}/references`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        target_node_id: selectedLinkNode.id,
                        link_text: linkText,
                        link_type: 'internal'
                    })
                });
            } catch (err) {
                console.error('Failed to create reference:', err);
            }
        }
    } else if (type === 'external') {
        linkUrl = document.getElementById('externalUrlInput').value;
        if (!linkUrl.startsWith('http')) {
            alert('External URL must start with http:// or https://');
            return;
        }
    } else if (type === 'veil') {
        linkUrl = document.getElementById('veilUriInput').value;
        if (!linkUrl.startsWith('veil://')) {
            alert('Veil URI must start with veil://');
            return;
        }
    }
    
    const markdown = `[${linkText}](${linkUrl})`;
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const text = editor.value;
    editor.value = text.substring(0, start) + markdown + text.substring(end);
    editor.selectionStart = editor.selectionEnd = start + markdown.length;
    editor.focus();
    
    closeModal('linkModal');
    selectedLinkNode = null;
    document.getElementById('linkSearchInput').value = '';
    document.getElementById('linkTextInput').value = '';
    document.getElementById('externalUrlInput').value = '';
    document.getElementById('veilUriInput').value = '';
    document.getElementById('linkType').value = 'internal';
    
    // Reload references if viewing current node
    if (currentNode) {
        loadReferences();
    }
}

// ====== ORIGINAL GIT CONFIG FUNCTION ======

async function saveGitConfigOriginal() {
    const repo = document.getElementById('gitRepo')?.value;
    if (!repo) {
        alert('Please enter a repository URL');
        return;
    }
    
    try {
        await fetch('/api/credentials', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key: 'git_repo', value: repo })
        });
        showToast('Git configuration saved', 'success');
    } catch (e) {
        console.error('Failed to save git config:', e);
        alert('Failed to save configuration');
    }
}

async function saveIPFSConfig() {
    const gateway = document.getElementById('ipfsGateway')?.value;
    if (!gateway) return;
    
    try {
        await fetch('/api/credentials', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key: 'ipfs_gateway', value: gateway })
        });
        showToast('IPFS configuration saved', 'success');
    } catch (e) {
        console.error('Failed to save IPFS config:', e);
        alert('Failed to save configuration');
    }
}

async function publishToIPFS() {
    if (!currentSite) {
        alert('Please select a site first');
        return;
    }
    
    try {
        showStatusBadge('Publishing to IPFS...', 'yellow');
        const resp = await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'ipfs',
                action: 'publish',
                payload: { site_id: currentSite.id }
            })
        });
        
        const result = await resp.json();
        showStatusBadge('Published to IPFS!', 'green');
        showToast(`Published! CID: ${result.cid || 'N/A'}`, 'success');
    } catch (e) {
        console.error('IPFS publish failed:', e);
        showStatusBadge('Publish failed', 'red');
        alert('Failed to publish to IPFS');
    }
}

async function saveNamecheapConfig() {
    const apiKey = document.getElementById('namecheapKey')?.value;
    const domain = document.getElementById('namecheapDomain')?.value;
    
    if (!apiKey || !domain) {
        alert('Please fill in all fields');
        return;
    }
    
    try {
        await fetch('/api/credentials', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key: 'namecheap_key', value: apiKey })
        });
        await fetch('/api/credentials', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key: 'namecheap_domain', value: domain })
        });
        showToast('Namecheap credentials saved', 'success');
    } catch (e) {
        console.error('Failed to save Namecheap config:', e);
        alert('Failed to save credentials');
    }
}


// Open preview in new tab
function openPreview() {
    if (!currentSite || !currentNode) {
        alert('Please select a note first');
        return;
    }
    window.open(`/preview/${currentSite.id}/${currentNode.id}`, '_blank');
}

// Open media upload
function openMediaUpload() {
    if (!currentNode) {
        alert('Please create or select a note first');
        return;
    }
    
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = 'image/*,video/*,audio/*';
    input.onchange = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        
        const formData = new FormData();
        formData.append('file', file);
        formData.append('node_id', currentNode.id);
        
        try {
            const response = await fetch('/api/media-upload', {
                method: 'POST',
                body: formData
            });
            
            if (response.ok) {
                const data = await response.json();
                const editor = document.getElementById('editor');
                const mediaMarkdown = `\n![${file.name}](${data.url})\n`;
                editor.value += mediaMarkdown;
                editor.dispatchEvent(new Event('input'));
                showToast('Media uploaded successfully');
            } else {
                showToast('Upload failed', 'error');
            }
        } catch (err) {
            console.error('Upload error:', err);
            showToast('Upload error', 'error');
        }
    };
    input.click();
}

// Show toast notification
function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    toast.className = `fixed bottom-4 right-4 px-6 py-3 rounded-lg shadow-lg text-white ${type === 'error' ? 'bg-red-600' : 'bg-green-600'} z-50`;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
}

// Format text
function formatText(format) {
    const editor = document.getElementById('editor');
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const selectedText = editor.value.substring(start, end);
    
    let formatted = '';
    switch(format) {
        case 'bold':
            formatted = `**${selectedText}**`;
            break;
        case 'italic':
            formatted = `*${selectedText}*`;
            break;
        case 'code':
            formatted = `\`${selectedText}\``;
            break;
    }
    
    editor.value = editor.value.substring(0, start) + formatted + editor.value.substring(end);
    editor.focus();
    editor.setSelectionRange(start + formatted.length, start + formatted.length);
    editor.dispatchEvent(new Event('input'));
}

// Hook up toolbar buttons
document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('boldBtn')?.addEventListener('click', () => formatText('bold'));
    document.getElementById('italicBtn')?.addEventListener('click', () => formatText('italic'));
    document.getElementById('codeBtn')?.addEventListener('click', () => formatText('code'));
    document.getElementById('linkBtn')?.addEventListener('click', () => openModal('linkModal'));
    document.getElementById('tagsBtn')?.addEventListener('click', () => openModal('tagsModal'));
    document.getElementById('mediaBtn')?.addEventListener('click', () => openMediaUpload());
    document.getElementById('versionsBtn')?.addEventListener('click', () => openModal('versionsModal'));
    document.getElementById('publishBtn')?.addEventListener('click', () => openModal('publishModal'));
    document.getElementById('saveBtn')?.addEventListener('click', () => saveCurrentNode());
});

