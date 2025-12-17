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
        <div class="site-item p-3 rounded-lg hover:bg-indigo-50 cursor-pointer transition border-l-4 ${
            site.id === currentSite?.id ? 'border-indigo-600 bg-indigo-50' : 'border-transparent'
        }" data-site-id="${site.id}" onclick="selectSite('${site.id}')" oncontextmenu="showSiteContextMenu(event, '${site.id}')">
            <div class="font-medium text-slate-900 text-sm">${site.name}</div>
            <div class="text-xs text-slate-500 mt-1">${site.description || 'No description'}</div>
        </div>
    `).join('');

    // Attach context menu event listeners
    document.querySelectorAll('.site-item').forEach((item) => {
        item.addEventListener('contextmenu', (e) => {
            const siteId = item.dataset.siteId;
            showSiteContextMenu(e, siteId);
        });
    });
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
        <div class="node-item p-3 rounded-lg hover:bg-indigo-50 cursor-pointer transition border-l-4 ${
            n.id === currentNode?.id ? 'border-indigo-600 bg-indigo-50' : 'border-transparent'
        }" data-node-id="${n.id}" onclick="openNode('${n.id}')" oncontextmenu="showNodeContextMenu(event, '${n.id}')">
            <div class="font-medium text-slate-900 text-sm flex items-center gap-2">
                <span class="text-xs px-2 py-0.5 bg-slate-200 rounded-full">${n.type || 'note'}</span>
                ${n.title || 'Untitled'}
            </div>
            <div class="text-xs text-slate-500 mt-1">${n.path}</div>
        </div>
    `).join('');

    // Attach context menu event listeners
    document.querySelectorAll('.node-item').forEach((item) => {
        item.addEventListener('contextmenu', (e) => {
            const nodeId = item.dataset.nodeId;
            showNodeContextMenu(e, nodeId);
        });
    });
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
            <div class="version-item p-3 bg-slate-50 rounded-lg border border-slate-200" data-version-id="${v.id}" oncontextmenu="showVersionContextMenu(event, '${v.id}')">
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

        // Attach context menu event listeners
        document.querySelectorAll('.version-item').forEach((item) => {
            item.addEventListener('contextmenu', (e) => {
                const versionId = item.dataset.versionId;
                showVersionContextMenu(e, versionId);
            });
        });
        
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
        if (!list) {
            console.error('pluginsList element not found');
            return;
        }
        list.innerHTML = '';
        (data || []).forEach(p => {
            const el = document.createElement('div');
            el.className = 'plugin-item flex items-center justify-between p-3 border rounded-lg';
            el.dataset.pluginSlug = p.slug;
            el.innerHTML = `<div class="text-sm"><div class="font-medium">${p.name}</div><div class="text-xs text-slate-500">${p.slug}</div></div><div><button class="toggle-btn px-3 py-1 rounded-lg text-sm">${p.enabled ? 'Disable' : 'Enable'}</button></div>`;
            el.addEventListener('contextmenu', (e) => showPluginContextMenu(e, p.slug));
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

async function createShader() {
    const shaderType = document.getElementById('shaderType').value;
    const name = prompt(`Enter ${shaderType} shader name:`, `New ${shaderType.charAt(0).toUpperCase() + shaderType.slice(1)} Shader`);

    if (!name) return;

    try {
        const response = await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'shader',
                action: 'create',
                payload: { type: shaderType, name }
            })
        });

        const result = await response.json();

        if (result.html) {
            // Create a new node with the shader content
            const nodeData = {
                type: 'shader-demo',
                title: name,
                content: result.html,
                path: `shaders/${name.toLowerCase().replace(/\s+/g, '-')}`,
                mime_type: 'text/html'
            };

            const createResponse = await fetch(`/api/sites/${currentSite.id}/nodes`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(nodeData)
            });

            if (createResponse.ok) {
                showToast(`${shaderType.charAt(0).toUpperCase() + shaderType.slice(1)} shader created successfully!`, 'success');
                await loadNodes(); // Refresh the node list
            } else {
                showToast('Failed to save shader', 'error');
            }
        }
    } catch (error) {
        console.error('Shader creation failed:', error);
        showToast('Failed to create shader', 'error');
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

// Context menu for plugins
let pluginContextMenu = null;

function createPluginContextMenu() {
    if (pluginContextMenu) {
        pluginContextMenu.remove();
    }

    pluginContextMenu = document.createElement('div');
    pluginContextMenu.className = 'fixed bg-white rounded-lg shadow-xl border border-slate-200 py-2 z-50 hidden';
    pluginContextMenu.style.minWidth = '200px';
    pluginContextMenu.innerHTML = `
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="enable">
            <i class="fas fa-toggle-on mr-2"></i>Enable
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="disable">
            <i class="fas fa-toggle-off mr-2"></i>Disable
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="configure">
            <i class="fas fa-cog mr-2"></i>Configure
        </div>
        <hr class="my-1">
        <div class="context-menu-item px-4 py-2 hover:bg-red-100 text-red-600 cursor-pointer text-sm" data-action="uninstall">
            <i class="fas fa-trash mr-2"></i>Uninstall
        </div>
    `;

    document.body.appendChild(pluginContextMenu);

    // Handle menu actions
    pluginContextMenu.querySelectorAll('.context-menu-item').forEach(item => {
        item.addEventListener('click', handlePluginContextMenuAction);
    });

    return pluginContextMenu;
}

function showPluginContextMenu(e, pluginSlug) {
    e.preventDefault();

    if (!pluginContextMenu) {
        createPluginContextMenu();
    }

    pluginContextMenu.dataset.pluginSlug = pluginSlug;
    pluginContextMenu.style.left = e.pageX + 'px';
    pluginContextMenu.style.top = e.pageY + 'px';
    pluginContextMenu.classList.remove('hidden');

    // Close on click outside
    setTimeout(() => {
        document.addEventListener('click', hidePluginContextMenu);
    }, 10);
}

function hidePluginContextMenu() {
    if (pluginContextMenu) {
        pluginContextMenu.classList.add('hidden');
    }
    document.removeEventListener('click', hidePluginContextMenu);
}

async function handlePluginContextMenuAction(e) {
    const action = e.currentTarget.dataset.action;
    const pluginSlug = pluginContextMenu.dataset.pluginSlug;

    hidePluginContextMenu();

    switch(action) {
        case 'enable':
            await togglePlugin({ slug: pluginSlug, enabled: false });
            break;
        case 'disable':
            await togglePlugin({ slug: pluginSlug, enabled: true });
            break;
        case 'configure':
            await configurePlugin(pluginSlug);
            break;
        case 'uninstall':
            if (confirm('Are you sure you want to uninstall this plugin?')) {
                await uninstallPlugin(pluginSlug);
            }
            break;
    }
}

async function configurePlugin(pluginSlug) {
    // Open configuration modal for the specific plugin
    alert(`Configuration for ${pluginSlug} - Feature coming soon!`);
}

async function uninstallPlugin(pluginSlug) {
    try {
        const response = await fetch(`/api/plugins/${pluginSlug}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            showToast('Plugin uninstalled');
            await loadPlugins();
        } else {
            showToast('Failed to uninstall plugin', 'error');
        }
    } catch (e) {
        console.error('Uninstall error:', e);
        showToast('Failed to uninstall plugin', 'error');
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
                <div class="media-item border border-slate-200 rounded-lg p-2 hover:border-indigo-500 cursor-pointer transition" data-media-id="${m.id}" data-media-url="${m.url}" data-media-filename="${m.filename}" onclick="insertMediaIntoEditor('${m.url}', '${m.filename}')" oncontextmenu="showMediaContextMenu(event, '${m.id}')">
                    ${isImage ? `<img src="${m.url}" alt="${m.filename}" class="w-full h-24 object-cover rounded mb-1">` : `<div class="w-full h-24 bg-slate-100 rounded mb-1 flex items-center justify-center"><i class="fas fa-file text-3xl text-slate-400"></i></div>`}
                    <p class="text-xs text-slate-600 truncate" title="${m.filename}">${m.filename}</p>
                    <p class="text-xs text-slate-400">${formatFileSize(m.size)}</p>
                </div>
            `;
        }).join('');

        // Attach context menu event listeners
        document.querySelectorAll('.media-item').forEach((item) => {
            item.addEventListener('contextmenu', (e) => {
                const mediaId = item.dataset.mediaId;
                showMediaContextMenu(e, mediaId);
            });
        });
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
    const editor = document.getElementById('noteContent') || document.getElementById('editor');
    
    if (!editor) {
        alert('Editor not found');
        return;
    }
    
    if (!linkText || linkText.trim() === '') {
        alert('Please provide link text');
        return;
    }
    
    let linkUrl = '';
    
    if (type === 'internal' && selectedLinkNode) {
        // Use relative path instead of veil:// for now - will be resolved by routing
        linkUrl = `/veil/note/${selectedLinkNode.id}`;
        
        // Create reference in database
        if (currentNode && currentSite) {
            try {
                await fetch(`/api/sites/${currentSite.id}/nodes/${currentNode.id}/references`, {
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
        if (!linkUrl || !linkUrl.startsWith('http')) {
            alert('External URL must start with http:// or https://');
            return;
        }
    } else if (type === 'veil') {
        linkUrl = document.getElementById('veilUriInput').value;
        if (!linkUrl || !linkUrl.startsWith('veil://')) {
            alert('Veil URI must start with veil://');
            return;
        }
    } else {
        alert('Please select a note to link to');
        return;
    }
    
    const markdown = `[${linkText}](${linkUrl})`;
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    
    editor.value = editor.value.substring(0, start) + markdown + editor.value.substring(end);
    editor.focus();
    editor.setSelectionRange(start + markdown.length, start + markdown.length);
    editor.dispatchEvent(new Event('input'));
    
    closeModal('linkModal');
    selectedLinkNode = null;
    document.getElementById('linkTextInput').value = '';
    document.getElementById('linkSearchInput').value = '';
    document.getElementById('linkSearchResults').innerHTML = '';
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
            
            const data = await response.json();
            
            if (response.ok && data.url) {
                const editor = document.getElementById('editor');
                const mediaMarkdown = `\n![${file.name}](${data.url})\n`;
                const start = editor.selectionStart;
                editor.value = editor.value.substring(0, start) + mediaMarkdown + editor.value.substring(start);
                editor.dispatchEvent(new Event('input'));
                showToast('Media uploaded successfully');
            } else {
                showToast('Upload failed: ' + (data.error || 'Unknown error'), 'error');
            }
        } catch (err) {
            console.error('Upload error:', err);
            showToast('Upload error: ' + err.message, 'error');
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


// Hook up preview button
document.addEventListener('DOMContentLoaded', function() {
    const previewBtn = document.getElementById('previewBtn');
    if (previewBtn) {
        previewBtn.addEventListener('click', openPreview);
    }
});

// Context menu system for nodes
let nodeContextMenu = null;

function createNodeContextMenu() {
    if (nodeContextMenu) {
        nodeContextMenu.remove();
    }

    nodeContextMenu = document.createElement('div');
    nodeContextMenu.className = 'fixed bg-white rounded-lg shadow-xl border border-slate-200 py-2 z-50 hidden';
    nodeContextMenu.style.minWidth = '200px';
    nodeContextMenu.innerHTML = `
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="preview">
            <i class="fas fa-eye mr-2"></i>Preview
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="copy-link">
            <i class="fas fa-link mr-2"></i>Copy Link
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="duplicate">
            <i class="fas fa-copy mr-2"></i>Duplicate
        </div>
        <hr class="my-1">
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="export">
            <i class="fas fa-download mr-2"></i>Export
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="publish">
            <i class="fas fa-rocket mr-2"></i>Publish
        </div>
        <hr class="my-1">
        <div class="context-menu-item px-4 py-2 hover:bg-red-100 text-red-600 cursor-pointer text-sm" data-action="delete">
            <i class="fas fa-trash mr-2"></i>Delete
        </div>
    `;

    document.body.appendChild(nodeContextMenu);

    // Handle menu actions
    nodeContextMenu.querySelectorAll('.context-menu-item').forEach(item => {
        item.addEventListener('click', handleNodeContextMenuAction);
    });

    return nodeContextMenu;
}

function showNodeContextMenu(e, nodeId) {
    e.preventDefault();

    if (!nodeContextMenu) {
        createNodeContextMenu();
    }

    nodeContextMenu.dataset.nodeId = nodeId;
    nodeContextMenu.style.left = e.pageX + 'px';
    nodeContextMenu.style.top = e.pageY + 'px';
    nodeContextMenu.classList.remove('hidden');

    // Close on click outside
    setTimeout(() => {
        document.addEventListener('click', hideNodeContextMenu);
    }, 10);
}

function hideNodeContextMenu() {
    if (nodeContextMenu) {
        nodeContextMenu.classList.add('hidden');
    }
    document.removeEventListener('click', hideNodeContextMenu);
}

async function handleNodeContextMenuAction(e) {
    const action = e.currentTarget.dataset.action;
    const nodeId = nodeContextMenu.dataset.nodeId;

    hideNodeContextMenu();

    switch(action) {
        case 'preview':
            if (currentSite) {
                window.open(`/preview/${currentSite.id}/${nodeId}`, '_blank');
            }
            break;
        case 'copy-link':
            if (currentSite) {
                const link = `/veil/note/${nodeId}`;
                navigator.clipboard.writeText(window.location.origin + link);
                showToast('Link copied to clipboard');
            }
            break;
        case 'duplicate':
            await duplicateNode(nodeId);
            break;
        case 'export':
            window.location.href = `/api/export?node_id=${nodeId}`;
            break;
        case 'publish':
            await publishNode(nodeId);
            break;
        case 'delete':
            if (confirm('Are you sure you want to delete this note?')) {
                await deleteNode(nodeId);
            }
            break;
    }
}

async function duplicateNode(nodeId) {
    try {
        const response = await fetch(`/api/sites/${currentSite.id}/nodes/${nodeId}`);
        const original = await response.json();

        const duplicate = {
            type: original.type,
            title: original.title + ' (Copy)',
            content: original.content,
            path: original.path.replace(/\.md$/, '-copy.md')
        };

        const createResponse = await fetch(`/api/sites/${currentSite.id}/nodes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(duplicate)
        });

        if (createResponse.ok) {
            showToast('Note duplicated');
            loadNodes();
        }
    } catch (err) {
        console.error('Duplicate error:', err);
        showToast('Failed to duplicate', 'error');
    }
}

async function publishNode(nodeId) {
    try {
        await fetch(`/api/sites/${currentSite.id}/nodes/${nodeId}/publish`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ visibility: 'public' })
        });
        showToast('Note published');
    } catch (err) {
        console.error('Publish error:', err);
        showToast('Failed to publish', 'error');
    }
}

async function deleteNode(nodeId) {
    try {
        await fetch(`/api/sites/${currentSite.id}/nodes/${nodeId}`, {
            method: 'DELETE'
        });
        showToast('Note deleted');
        if (currentNode && currentNode.id === nodeId) {
            currentNode = null;
            document.getElementById('editor').value = '';
        }
        loadNodes();
    } catch (err) {
        console.error('Delete error:', err);
        showToast('Failed to delete', 'error');
    }
}

// Context menu system for sites
let siteContextMenu = null;

function createSiteContextMenu() {
    if (siteContextMenu) {
        siteContextMenu.remove();
    }

    siteContextMenu = document.createElement('div');
    siteContextMenu.className = 'fixed bg-white rounded-lg shadow-xl border border-slate-200 py-2 z-50 hidden';
    siteContextMenu.style.minWidth = '200px';
    siteContextMenu.innerHTML = `
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="open">
            <i class="fas fa-folder-open mr-2"></i>Open
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="edit">
            <i class="fas fa-edit mr-2"></i>Edit
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="duplicate">
            <i class="fas fa-copy mr-2"></i>Duplicate
        </div>
        <hr class="my-1">
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="export">
            <i class="fas fa-download mr-2"></i>Export
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="publish">
            <i class="fas fa-rocket mr-2"></i>Publish All
        </div>
        <hr class="my-1">
        <div class="context-menu-item px-4 py-2 hover:bg-red-100 text-red-600 cursor-pointer text-sm" data-action="delete">
            <i class="fas fa-trash mr-2"></i>Delete
        </div>
    `;

    document.body.appendChild(siteContextMenu);

    // Handle menu actions
    siteContextMenu.querySelectorAll('.context-menu-item').forEach(item => {
        item.addEventListener('click', handleSiteContextMenuAction);
    });

    return siteContextMenu;
}

function showSiteContextMenu(e, siteId) {
    e.preventDefault();

    if (!siteContextMenu) {
        createSiteContextMenu();
    }

    siteContextMenu.dataset.siteId = siteId;
    siteContextMenu.style.left = e.pageX + 'px';
    siteContextMenu.style.top = e.pageY + 'px';
    siteContextMenu.classList.add('hidden');

    // Close on click outside
    setTimeout(() => {
        document.addEventListener('click', hideSiteContextMenu);
    }, 10);
}

function hideSiteContextMenu() {
    if (siteContextMenu) {
        siteContextMenu.classList.add('hidden');
    }
    document.removeEventListener('click', hideSiteContextMenu);
}

async function handleSiteContextMenuAction(e) {
    const action = e.currentTarget.dataset.action;
    const siteId = siteContextMenu.dataset.siteId;

    hideSiteContextMenu();

    switch(action) {
        case 'open':
            await selectSite(siteId);
            break;
        case 'edit':
            await editSite(siteId);
            break;
        case 'duplicate':
            await duplicateSite(siteId);
            break;
        case 'export':
            window.location.href = `/api/export?site_id=${siteId}`;
            break;
        case 'publish':
            await publishSite(siteId);
            break;
        case 'delete':
            if (confirm('Are you sure you want to delete this site?')) {
                await deleteSite(siteId);
            }
            break;
    }
}

async function editSite(siteId) {
    const site = allSites.find(s => s.id === siteId);
    if (!site) return;

    // Populate site editor modal
    document.getElementById('siteNameInput').value = site.name || '';
    document.getElementById('siteDescriptionInput').value = site.description || '';
    document.getElementById('siteTypeInput').value = site.type || 'project';

    // Store site ID for update
    document.getElementById('siteEditorModal').dataset.editSiteId = siteId;

    // Change button text
    document.querySelector('#siteEditorModal button[id="saveSiteBtn"]').textContent = 'Update Site';
    document.querySelector('#siteEditorModal button[id="saveSiteBtn"]').onclick = () => updateSite(siteId);

    openModal('siteEditorModal');
}

async function updateSite(siteId) {
    const name = document.getElementById('siteNameInput').value;
    const description = document.getElementById('siteDescriptionInput').value;
    const type = document.getElementById('siteTypeInput').value;

    try {
        const resp = await fetch('/api/sites', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: siteId, name, description, type })
        });

        if (!resp.ok) {
            const body = await resp.text().catch(() => resp.statusText);
            alert('Failed to update site: ' + body);
            return;
        }

        // Refresh sites list
        await loadSites();

        // Reset modal
        delete document.getElementById('siteEditorModal').dataset.editSiteId;
        document.querySelector('#siteEditorModal button[id="saveSiteBtn"]').textContent = 'Save';
        document.querySelector('#siteEditorModal button[id="saveSiteBtn"]').onclick = saveSiteEdits;

        closeModal('siteEditorModal');
        showToast('Site updated!', 'success');
    } catch (e) {
        console.error('Failed to update site', e);
        showToast('Failed to update site', 'error');
    }
}

async function duplicateSite(siteId) {
    const site = allSites.find(s => s.id === siteId);
    if (!site) return;

    const name = prompt('New site name:', site.name + ' (Copy)');
    if (!name) return;

    try {
        const resp = await fetch('/api/sites', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name,
                description: site.description,
                type: site.type
            })
        });

        if (resp.ok) {
            const newSite = await resp.json();
            allSites.push(newSite);
            renderSitesList();
            showToast('Site duplicated!', 'success');
        } else {
            showToast('Failed to duplicate site', 'error');
        }
    } catch (e) {
        console.error('Duplicate site error:', e);
        showToast('Failed to duplicate site', 'error');
    }
}

async function publishSite(siteId) {
    try {
        await fetch(`/api/sites/${siteId}/publish`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ visibility: 'public' })
        });
        showToast('Site published');
    } catch (err) {
        console.error('Publish site error:', err);
        showToast('Failed to publish site', 'error');
    }
}

async function deleteSite(siteId) {
    try {
        await fetch(`/api/sites/${siteId}`, {
            method: 'DELETE'
        });
        showToast('Site deleted');
        await loadSites();
        if (currentSite && currentSite.id === siteId) {
            currentSite = null;
            currentNode = null;
            document.getElementById('editor').value = '';
            document.getElementById('breadcrumb').innerHTML = '<span>Veil</span>';
        }
    } catch (err) {
        console.error('Delete site error:', err);
        showToast('Failed to delete site', 'error');
    }
}

// ====== TERMINAL SCRIPTING FUNCTIONS ======

function openTerminalModal() {
    // Reset form
    document.getElementById('terminalActionSelect').value = 'execute';
    document.getElementById('terminalWorkingDir').value = './';
    document.getElementById('terminalCommand').value = '';
    document.getElementById('terminalEnv').value = '';
    document.getElementById('terminalOutput').innerHTML = '<div class="text-slate-500">Ready to execute commands...</div>';

    // Show execute fields by default
    showTerminalFields('execute');

    // Hook up action change listener
    document.getElementById('terminalActionSelect').addEventListener('change', function(e) {
        showTerminalFields(e.target.value);
    });

    openModal('terminalModal');
}

function showTerminalFields(action) {
    // Hide all field groups
    const fieldGroups = ['executeFields', 'installFields', 'generateFields', 'testFields', 'buildFields', 'dependenciesFields'];
    fieldGroups.forEach(id => {
        document.getElementById(id).classList.add('hidden');
    });

    // Show relevant fields
    switch(action) {
        case 'execute':
            document.getElementById('executeFields').classList.remove('hidden');
            break;
        case 'install_package':
            document.getElementById('installFields').classList.remove('hidden');
            break;
        case 'generate_code':
            document.getElementById('generateFields').classList.remove('hidden');
            break;
        case 'run_tests':
            document.getElementById('testFields').classList.remove('hidden');
            break;
        case 'build_project':
            document.getElementById('buildFields').classList.remove('hidden');
            break;
        case 'check_dependencies':
            document.getElementById('dependenciesFields').classList.remove('hidden');
            break;
    }
}

async function executeTerminalAction() {
    const action = document.getElementById('terminalActionSelect').value;
    const workingDir = document.getElementById('terminalWorkingDir').value || './';

    const outputEl = document.getElementById('terminalOutput');
    outputEl.innerHTML = '<div class="text-yellow-400">Executing...</div>';

    try {
        let payload = { working_directory: workingDir };

        switch(action) {
            case 'execute':
                payload.command = document.getElementById('terminalCommand').value;
                if (document.getElementById('terminalEnv').value.trim()) {
                    const envLines = document.getElementById('terminalEnv').value.trim().split('\n');
                    const env = {};
                    envLines.forEach(line => {
                        const [key, ...valueParts] = line.split('=');
                        if (key && valueParts.length > 0) {
                            env[key.trim()] = valueParts.join('=').trim();
                        }
                    });
                    payload.environment = env;
                }
                break;

            case 'install_package':
                payload.manager = document.getElementById('packageManager').value;
                payload.packages = document.getElementById('packageNames').value.split(',').map(p => p.trim()).filter(p => p);
                break;

            case 'generate_code':
                payload.template = document.getElementById('codeTemplate').value;
                payload.name = document.getElementById('codeName').value;
                payload.directory = document.getElementById('codeDirectory').value;
                break;

            case 'run_tests':
                payload.type = document.getElementById('testProjectType').value;
                break;

            case 'build_project':
                payload.type = document.getElementById('buildProjectType').value;
                break;

            case 'check_dependencies':
                payload.dependencies = document.getElementById('dependenciesList').value.split(',').map(d => d.trim()).filter(d => d);
                break;
        }

        const response = await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'terminal',
                action: action,
                payload: payload
            })
        });

        const result = await response.json();

        if (response.ok && result) {
            displayTerminalResult(result);
        } else {
            outputEl.innerHTML = `<div class="text-red-400">Error: ${result?.error || 'Unknown error'}</div>`;
        }

    } catch (error) {
        console.error('Terminal execution error:', error);
        outputEl.innerHTML = `<div class="text-red-400">Error: ${error.message}</div>`;
    }
}

function displayTerminalResult(result) {
    const outputEl = document.getElementById('terminalOutput');

    if (result.success) {
        let html = '<div class="text-green-400 mb-2">✓ Command executed successfully</div>';

        if (result.output) {
            html += `<div class="text-slate-300 whitespace-pre-wrap">${escapeHtml(result.output)}</div>`;
        }

        if (result.files) {
            html += '<div class="text-blue-400 mt-2">Generated files:</div>';
            Object.keys(result.files).forEach(filename => {
                html += `<div class="text-slate-300 ml-2">• ${filename}</div>`;
            });
        }

        if (result.dependencies) {
            html += '<div class="text-blue-400 mt-2">Dependencies check:</div>';
            Object.entries(result.dependencies).forEach(([dep, installed]) => {
                const status = installed ? '<span class="text-green-400">✓</span>' : '<span class="text-red-400">✗</span>';
                html += `<div class="text-slate-300 ml-2">${status} ${dep}</div>`;
            });
        }

        outputEl.innerHTML = html;
    } else {
        let html = '<div class="text-red-400 mb-2">✗ Command failed</div>';

        if (result.error) {
            html += `<div class="text-red-300">${escapeHtml(result.error)}</div>`;
        }

        if (result.output) {
            html += `<div class="text-slate-300 whitespace-pre-wrap">${escapeHtml(result.output)}</div>`;
        }

        if (result.exit_code !== undefined) {
            html += `<div class="text-yellow-400">Exit code: ${result.exit_code}</div>`;
        }

        outputEl.innerHTML = html;
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ====== REMINDER SYSTEM FUNCTIONS ======

function openReminderModal() {
    openModal('reminderModal');
    loadReminders();
}

async function loadReminders() {
    try {
        const resp = await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'reminder',
                action: 'list',
                payload: {}
            })
        });

        const reminders = await resp.json();
        const list = document.getElementById('remindersList');

        if (!reminders || reminders.length === 0) {
            list.innerHTML = '<div class="text-slate-500 text-sm">No reminders yet</div>';
            return;
        }

        list.innerHTML = reminders.map(r => `
            <div class="reminder-item p-3 bg-slate-50 rounded-lg border border-slate-200 hover:bg-slate-100 transition cursor-pointer" data-reminder-id="${r.id}">
                <div class="flex justify-between items-start">
                    <div>
                        <div class="font-medium text-sm">${r.title}</div>
                        <div class="text-xs text-slate-500">${new Date(r.remind_at * 1000).toLocaleString()}</div>
                        <div class="text-xs text-slate-600 mt-1">${r.description || ''}</div>
                        <div class="text-xs text-slate-400 mt-1">Status: ${r.status} | Recurrence: ${r.recurrence}</div>
                    </div>
                    <div class="flex gap-2">
                        <button onclick="dismissReminder('${r.id}')" class="px-2 py-1 bg-green-600 text-white text-xs rounded hover:bg-green-700">Dismiss</button>
                        <button onclick="snoozeReminder('${r.id}')" class="px-2 py-1 bg-yellow-600 text-white text-xs rounded hover:bg-yellow-700">Snooze</button>
                    </div>
                </div>
            </div>
        `).join('');

        // Attach context menu event listeners
        document.querySelectorAll('.reminder-item').forEach((item, index) => {
            item.addEventListener('contextmenu', (e) => showReminderContextMenu(e, reminders[index].id));
        });
    } catch (e) {
        console.error('Failed to load reminders:', e);
        document.getElementById('remindersList').innerHTML = '<div class="text-red-500 text-sm">Failed to load reminders</div>';
    }
}

async function createReminder() {
    const title = document.getElementById('reminderTitle').value;
    const dateTime = document.getElementById('reminderDateTime').value;
    const description = document.getElementById('reminderDescription').value;
    const recurrence = document.getElementById('reminderRecurrence').value;

    if (!title || !dateTime) {
        alert('Title and date/time are required');
        return;
    }

    const remindAt = new Date(dateTime).getTime() / 1000;

    try {
        await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'reminder',
                action: 'create',
                payload: {
                    title,
                    description,
                    remind_at: remindAt,
                    recurrence
                }
            })
        });

        // Clear form
        document.getElementById('reminderTitle').value = '';
        document.getElementById('reminderDateTime').value = '';
        document.getElementById('reminderDescription').value = '';
        document.getElementById('reminderRecurrence').value = 'none';

        loadReminders();
        showToast('Reminder created!', 'success');
    } catch (e) {
        console.error('Failed to create reminder:', e);
        showToast('Failed to create reminder', 'error');
    }
}

async function dismissReminder(id) {
    try {
        await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'reminder',
                action: 'dismiss',
                payload: { id }
            })
        });

        loadReminders();
        showToast('Reminder dismissed', 'success');
    } catch (e) {
        console.error('Failed to dismiss reminder:', e);
        showToast('Failed to dismiss reminder', 'error');
    }
}

async function snoozeReminder(id, minutes = null) {
    if (minutes === null) {
        minutes = prompt('Snooze for how many minutes?', '15');
        if (!minutes || isNaN(parseInt(minutes))) return;
    }

    try {
        await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'reminder',
                action: 'snooze',
                payload: { id, snooze_minutes: parseInt(minutes) }
            })
        });

        loadReminders();
        showToast(`Reminder snoozed for ${minutes} minutes`, 'success');
    } catch (e) {
        console.error('Failed to snooze reminder:', e);
        showToast('Failed to snooze reminder', 'error');
    }
}

// ====== REMINDER CONTEXT MENU FUNCTIONS ======

let reminderContextMenu = null;

function createReminderContextMenu() {
    if (reminderContextMenu) {
        reminderContextMenu.remove();
    }

    reminderContextMenu = document.createElement('div');
    reminderContextMenu.className = 'fixed bg-white rounded-lg shadow-xl border border-slate-200 py-2 z-50 hidden';
    reminderContextMenu.style.minWidth = '200px';
    reminderContextMenu.innerHTML = `
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="dismiss">
            <i class="fas fa-check mr-2"></i>Dismiss
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="snooze-15">
            <i class="fas fa-clock mr-2"></i>Snooze 15 min
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="snooze-30">
            <i class="fas fa-clock mr-2"></i>Snooze 30 min
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="snooze-60">
            <i class="fas fa-clock mr-2"></i>Snooze 1 hour
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="snooze-custom">
            <i class="fas fa-clock mr-2"></i>Snooze Custom...
        </div>
        <hr class="my-1">
        <div class="context-menu-item px-4 py-2 hover:bg-slate-100 cursor-pointer text-sm" data-action="edit">
            <i class="fas fa-edit mr-2"></i>Edit
        </div>
        <div class="context-menu-item px-4 py-2 hover:bg-red-100 text-red-600 cursor-pointer text-sm" data-action="delete">
            <i class="fas fa-trash mr-2"></i>Delete
        </div>
    `;

    document.body.appendChild(reminderContextMenu);

    // Handle menu actions
    reminderContextMenu.querySelectorAll('.context-menu-item').forEach(item => {
        item.addEventListener('click', handleReminderContextMenuAction);
    });

    return reminderContextMenu;
}

function showReminderContextMenu(e, reminderId) {
    e.preventDefault();

    if (!reminderContextMenu) {
        createReminderContextMenu();
    }

    reminderContextMenu.dataset.reminderId = reminderId;
    reminderContextMenu.style.left = e.pageX + 'px';
    reminderContextMenu.style.top = e.pageY + 'px';
    reminderContextMenu.classList.remove('hidden');

    // Close on click outside
    setTimeout(() => {
        document.addEventListener('click', hideReminderContextMenu);
    }, 10);
}

function hideReminderContextMenu() {
    if (reminderContextMenu) {
        reminderContextMenu.classList.add('hidden');
    }
    document.removeEventListener('click', hideReminderContextMenu);
}

async function handleReminderContextMenuAction(e) {
    const action = e.currentTarget.dataset.action;
    const reminderId = reminderContextMenu.dataset.reminderId;

    hideReminderContextMenu();

    switch(action) {
        case 'dismiss':
            await dismissReminder(reminderId);
            break;
        case 'snooze-15':
            await snoozeReminder(reminderId, 15);
            break;
        case 'snooze-30':
            await snoozeReminder(reminderId, 30);
            break;
        case 'snooze-60':
            await snoozeReminder(reminderId, 60);
            break;
        case 'snooze-custom':
            await snoozeReminder(reminderId);
            break;
        case 'edit':
            await editReminder(reminderId);
            break;
        case 'delete':
            if (confirm('Are you sure you want to delete this reminder?')) {
                await deleteReminder(reminderId);
            }
            break;
    }
}

async function editReminder(id) {
    // Load reminder details and populate form for editing
    try {
        const resp = await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'reminder',
                action: 'get',
                payload: { id }
            })
        });

        const reminder = await resp.json();
        if (reminder) {
            // Populate form with existing data
            document.getElementById('reminderTitle').value = reminder.title || '';
            document.getElementById('reminderDescription').value = reminder.description || '';
            document.getElementById('reminderRecurrence').value = reminder.recurrence || 'none';

            // Convert timestamp to datetime-local format
            const date = new Date(reminder.remind_at * 1000);
            const datetimeLocal = date.toISOString().slice(0, 16);
            document.getElementById('reminderDateTime').value = datetimeLocal;

            // Store reminder ID for update
            document.getElementById('reminderModal').dataset.editId = id;

            // Change button text
            document.querySelector('#reminderModal button[onclick*="createReminder"]').textContent = 'Update Reminder';
            document.querySelector('#reminderModal button[onclick*="createReminder"]').onclick = () => updateReminder(id);

            openModal('reminderModal');
        }
    } catch (e) {
        console.error('Failed to load reminder for editing:', e);
        showToast('Failed to load reminder', 'error');
    }
}

async function updateReminder(id) {
    const title = document.getElementById('reminderTitle').value;
    const dateTime = document.getElementById('reminderDateTime').value;
    const description = document.getElementById('reminderDescription').value;
    const recurrence = document.getElementById('reminderRecurrence').value;

    if (!title || !dateTime) {
        alert('Title and date/time are required');
        return;
    }

    const remindAt = new Date(dateTime).getTime() / 1000;

    try {
        await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'reminder',
                action: 'update',
                payload: {
                    id,
                    title,
                    description,
                    remind_at: remindAt,
                    recurrence
                }
            })
        });

        // Clear form and reset
        document.getElementById('reminderTitle').value = '';
        document.getElementById('reminderDateTime').value = '';
        document.getElementById('reminderDescription').value = '';
        document.getElementById('reminderRecurrence').value = 'none';
        delete document.getElementById('reminderModal').dataset.editId;

        // Reset button
        document.querySelector('#reminderModal button[onclick*="updateReminder"]').textContent = 'Create Reminder';
        document.querySelector('#reminderModal button[onclick*="updateReminder"]').onclick = createReminder;

        loadReminders();
        closeModal('reminderModal');
        showToast('Reminder updated!', 'success');
    } catch (e) {
        console.error('Failed to update reminder:', e);
        showToast('Failed to update reminder', 'error');
    }
}

async function deleteReminder(id) {
    try {
        await fetch('/api/plugin-execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                plugin: 'reminder',
                action: 'delete',
                payload: { id }
            })
        });

        loadReminders();
        showToast('Reminder deleted', 'success');
    } catch (e) {
        console.error('Failed to delete reminder:', e);
        showToast('Failed to delete reminder', 'error');
    }
}

// ====== CODEX EMBED HANDLERS ======
function showCodexPanel() {
    const panel = document.getElementById('codexPanel');
    if (!panel) return;
    panel.classList.remove('hidden');
    // ensure iframe has src set (lazy load)
    const iframe = document.getElementById('codexIframe');
    if (iframe && !iframe.src) {
        iframe.src = '/codex-prototype/index.html';
    }
}

function hideCodexPanel() {
    const panel = document.getElementById('codexPanel');
    if (!panel) return;
    panel.classList.add('hidden');
}

// Send selected node data to Codex iframe (postMessage)
function sendNodeToCodex(node) {
    try {
        const iframe = document.getElementById('codexIframe');
        if (!iframe) return;
        iframe.contentWindow.postMessage({ type: 'veil:node-selected', node }, '*');
    } catch (e) {
        console.warn('Failed to post message to Codex iframe', e);
    }
}

// Hook the buttons
(function attachCodexHandlers() {
    const codexBtn = document.getElementById('codexBtn');
    if (codexBtn) codexBtn.addEventListener('click', () => showCodexPanel());
    const codexClose = document.getElementById('codexCloseBtn');
    if (codexClose) codexClose.addEventListener('click', () => hideCodexPanel());

    // When a node is opened in Veil, send to Codex if panel open
    const originalOpenNode = window.openNode;
    window.openNode = async function (nodeId) {
        await originalOpenNode(nodeId);
        if (!document.getElementById('codexPanel')) return;
        if (!document.getElementById('codexPanel').classList.contains('hidden')) {
            // currentNode is set by openNode; send its minimal data
            if (window.currentNode) sendNodeToCodex({ id: currentNode.id, title: currentNode.title, content: currentNode.content });
        }
    }
})();
