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
    document.getElementById('pluginsBtn')?.addEventListener('click', async () => { openModal('pluginsModal'); await loadPlugins(); });
    document.getElementById('openSiteEditor')?.addEventListener('click', () => { openModal('siteEditorModal'); populateSiteEditor(); });
    document.getElementById('saveSiteBtn')?.addEventListener('click', saveSiteEdits);
    document.getElementById('addUriBtn')?.addEventListener('click', addUriPrompt);
    document.getElementById('globalSearch')?.addEventListener('input', (e) => filterNodes(e.target.value));
    
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
