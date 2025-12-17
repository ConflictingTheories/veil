// Codex Universal PWA - Main Application Logic

class CodexApp {
    constructor() {
        this.db = null;
        this.currentRepo = 'urn:codex:local/my-collection';
        this.data = {
            texts: [],
            entities: [],
            annotations: [],
            commits: []
        };
        this.init();
    }

    async init() {
        await this.initDatabase();
        await this.loadData();
        this.setupEventListeners();
        this.renderAll();
    }

    // IndexedDB Setup for Offline Storage
    async initDatabase() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open('CodexUniversalDB', 1);

            request.onerror = () => reject(request.error);
            request.onsuccess = () => {
                this.db = request.result;
                resolve();
            };

            request.onupgradeneeded = (event) => {
                const db = event.target.result;

                // Create object stores
                if (!db.objectStoreNames.contains('texts')) {
                    const textsStore = db.createObjectStore('texts', { keyPath: 'urn' });
                    textsStore.createIndex('language', 'language', { unique: false });
                }

                if (!db.objectStoreNames.contains('entities')) {
                    const entitiesStore = db.createObjectStore('entities', { keyPath: 'urn' });
                    entitiesStore.createIndex('type', 'type', { unique: false });
                }

                if (!db.objectStoreNames.contains('annotations')) {
                    const annotationsStore = db.createObjectStore('annotations', { keyPath: 'id', autoIncrement: true });
                    annotationsStore.createIndex('textUrn', 'textUrn', { unique: false });
                    annotationsStore.createIndex('entityUrn', 'entityUrn', { unique: false });
                }

                if (!db.objectStoreNames.contains('commits')) {
                    const commitsStore = db.createObjectStore('commits', { keyPath: 'hash' });
                    commitsStore.createIndex('timestamp', 'timestamp', { unique: false });
                }
            };
        });
    }

    // Load data from IndexedDB
    async loadData() {
        this.data.texts = await this.getAllFromStore('texts');
        this.data.entities = await this.getAllFromStore('entities');
        this.data.annotations = await this.getAllFromStore('annotations');
        this.data.commits = await this.getAllFromStore('commits');

        // Load sample data if empty
        if (this.data.texts.length === 0) {
            await this.loadSampleData();
        }
    }

    // Helper: Get all items from a store
    getAllFromStore(storeName) {
        return new Promise((resolve, reject) => {
            const transaction = this.db.transaction([storeName], 'readonly');
            const store = transaction.objectStore(storeName);
            const request = store.getAll();

            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    // Helper: Add item to store
    addToStore(storeName, item) {
        return new Promise((resolve, reject) => {
            const transaction = this.db.transaction([storeName], 'readwrite');
            const store = transaction.objectStore(storeName);
            const request = store.add(item);

            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    // Load Sample Data
    async loadSampleData() {
        const sampleTexts = [
            {
                urn: 'urn:codex:iliad@homer:book1:lines1-7',
                content: 'μῆνιν ἄειδε θεὰ Πηληϊάδεω Ἀχιλῆος οὐλομένην, ἣ μυρί᾽ Ἀχαιοῖς ἄλγε᾽ ἔθηκε',
                language: 'grc',
                metadata: {
                    work: 'Iliad',
                    author: 'Homer',
                    book: 1,
                    translation: 'Sing, O goddess, the anger of Achilles son of Peleus, that brought countless ills upon the Achaeans'
                }
            },
            {
                urn: 'urn:codex:odyssey@homer:book1:lines1-10',
                content: 'Ἄνδρα μοι ἔννεπε, Μοῦσα, πολύτροπον, ὃς μάλα πολλὰ πλάγχθη',
                language: 'grc',
                metadata: {
                    work: 'Odyssey',
                    author: 'Homer',
                    book: 1,
                    translation: 'Tell me, O Muse, of the man of many ways, who wandered far and wide'
                }
            },
            {
                urn: 'urn:codex:republic@plato:book1:excerpt',
                content: 'κατέβην χθὲς εἰς Πειραιᾶ μετὰ Γλαύκωνος τοῦ Ἀρίστωνος',
                language: 'grc',
                metadata: {
                    work: 'Republic',
                    author: 'Plato',
                    book: 1,
                    translation: 'I went down yesterday to the Piraeus with Glaucon the son of Ariston'
                }
            }
        ];

        const sampleEntities = [
            {
                urn: 'urn:codex:entity/achilles',
                type: 'Character',
                labels: {
                    en: 'Achilles',
                    grc: 'Ἀχιλλεύς'
                },
                properties: {
                    epithets: ['Swift-footed', 'Son of Peleus', 'Godlike'],
                    parent: 'Peleus',
                    mother: 'Thetis',
                    role: 'Hero of the Trojan War'
                }
            },
            {
                urn: 'urn:codex:entity/odysseus',
                type: 'Character',
                labels: {
                    en: 'Odysseus',
                    grc: 'Ὀδυσσεύς'
                },
                properties: {
                    epithets: ['Man of many ways', 'Polytropos', 'Cunning'],
                    father: 'Laertes',
                    kingdom: 'Ithaca',
                    role: 'King and hero'
                }
            },
            {
                urn: 'urn:codex:entity/troy',
                type: 'Place',
                labels: {
                    en: 'Troy',
                    grc: 'Τροία'
                },
                properties: {
                    region: 'Asia Minor',
                    significance: 'Site of the Trojan War',
                    alternative_names: ['Ilium', 'Ilion']
                }
            },
            {
                urn: 'urn:codex:entity/socrates',
                type: 'Character',
                labels: {
                    en: 'Socrates',
                    grc: 'Σωκράτης'
                },
                properties: {
                    role: 'Philosopher',
                    student_of: 'Various sophists',
                    teacher_of: 'Plato',
                    era: 'Classical Athens'
                }
            }
        ];

        const sampleAnnotations = [
            {
                textUrn: 'urn:codex:iliad@homer:book1:lines1-7',
                entityUrn: 'urn:codex:entity/achilles',
                start: 24,
                end: 32,
                role: 'subject',
                certainty: 0.98,
                contributor: 'local-user',
                timestamp: Date.now()
            },
            {
                textUrn: 'urn:codex:odyssey@homer:book1:lines1-10',
                entityUrn: 'urn:codex:entity/odysseus',
                start: 0,
                end: 20,
                role: 'subject',
                certainty: 0.95,
                contributor: 'local-user',
                timestamp: Date.now()
            }
        ];

        // Add to database
        for (const text of sampleTexts) {
            await this.addToStore('texts', text);
        }

        for (const entity of sampleEntities) {
            await this.addToStore('entities', entity);
        }

        for (const annotation of sampleAnnotations) {
            await this.addToStore('annotations', annotation);
        }

        // Reload data
        await this.loadData();
    }

    // Event Listeners
    setupEventListeners() {
        // Navigation tabs
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                this.switchTab(e.target.dataset.panel);
            });
        });
    }

    // Switch tabs
    switchTab(panelId) {
        // Update tabs
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.classList.remove('active');
            if (tab.dataset.panel === panelId) {
                tab.classList.add('active');
            }
        });

        // Update panels
        document.querySelectorAll('.content-panel').forEach(panel => {
            panel.classList.remove('active');
        });
        document.getElementById(panelId).classList.add('active');
    }

    // Render all data
    renderAll() {
        this.renderRecentActivity();
        this.renderTexts();
        this.renderEntities();
        this.renderAnnotations();
        this.renderTrustedAuthorities();
    }

    // Render Recent Activity
    renderRecentActivity() {
        const container = document.getElementById('recentActivity');
        const recentItems = [
            ...this.data.texts.slice(0, 2).map(t => ({ ...t, type: 'text' })),
            ...this.data.entities.slice(0, 2).map(e => ({ ...e, type: 'entity' })),
            ...this.data.annotations.slice(0, 2).map(a => ({ ...a, type: 'annotation' }))
        ];

        container.innerHTML = recentItems.map(item => this.renderCard(item)).join('');
    }

    // Render Texts
    renderTexts() {
        const container = document.getElementById('textsGrid');
        container.innerHTML = this.data.texts.map(text => this.renderCard({ ...text, type: 'text' })).join('');
    }

    // Render Entities
    renderEntities() {
        const container = document.getElementById('entitiesGrid');
        container.innerHTML = this.data.entities.map(entity => this.renderCard({ ...entity, type: 'entity' })).join('');
    }

    // Render Annotations
    renderAnnotations() {
        const container = document.getElementById('annotationsGrid');
        container.innerHTML = this.data.annotations.map(annotation => this.renderCard({ ...annotation, type: 'annotation' })).join('');
    }

    // Render Card
    renderCard(item) {
        if (item.type === 'text') {
            return `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">${item.metadata?.work || 'Untitled'}</div>
                            <div class="card-meta">${item.urn}</div>
                        </div>
                        <span class="card-type text">Text</span>
                    </div>
                    <div class="card-content preview">
                        <p><strong>Original:</strong> ${item.content}</p>
                        ${item.metadata?.translation ? `<p><strong>Translation:</strong> ${item.metadata.translation}</p>` : ''}
                    </div>
                    <div class="action-buttons">
                        <button class="btn" onclick="app.viewItem('${item.urn}', 'text')">View</button>
                        <button class="btn secondary" onclick="app.editItem('${item.urn}', 'text')">Edit</button>
                    </div>
                </div>
            `;
        } else if (item.type === 'entity') {
            const epithetsList = Array.isArray(item.properties?.epithets) 
                ? item.properties.epithets.join(', ') 
                : '';
            return `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">${item.labels?.en || 'Unnamed Entity'}</div>
                            <div class="card-meta">${item.urn}</div>
                        </div>
                        <span class="card-type entity">${item.type}</span>
                    </div>
                    <div class="card-content">
                        ${epithetsList ? `<p><strong>Epithets:</strong> ${epithetsList}</p>` : ''}
                        ${item.properties?.role ? `<p><strong>Role:</strong> ${item.properties.role}</p>` : ''}
                    </div>
                    <div class="action-buttons">
                        <button class="btn" onclick="app.viewItem('${item.urn}', 'entity')">View</button>
                        <button class="btn secondary" onclick="app.editItem('${item.urn}', 'entity')">Edit</button>
                    </div>
                </div>
            `;
        } else if (item.type === 'annotation') {
            return `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <div class="card-title">Annotation</div>
                            <div class="card-meta">Text: ${item.textUrn}</div>
                        </div>
                        <span class="card-type annotation">Link</span>
                    </div>
                    <div class="card-content">
                        <p><strong>Entity:</strong> ${item.entityUrn}</p>
                        <p><strong>Role:</strong> ${item.role}</p>
                        <p><strong>Certainty:</strong> ${(item.certainty * 100).toFixed(0)}%</p>
                    </div>
                    <div class="action-buttons">
                        <button class="btn" onclick="app.viewItem(${item.id}, 'annotation')">View</button>
                    </div>
                </div>
            `;
        }
    }

    // Render Trusted Authorities
    renderTrustedAuthorities() {
        const container = document.getElementById('trustedAuthorities');
        const authorities = [
            { name: 'Perseus Digital Library', url: 'https://codex.perseus.org', status: 'synced' },
            { name: 'Vatican Apostolic Archive', url: 'https://codex.vatican.va', status: 'synced' },
            { name: 'Oxford Text Archive', url: 'https://codex.ox.ac.uk', status: 'pending' }
        ];

        container.innerHTML = authorities.map(auth => `
            <div style="display: flex; justify-content: space-between; align-items: center; padding: 1rem; background: rgba(255,255,255,0.5); border-radius: 6px; margin-bottom: 0.5rem;">
                <div>
                    <div style="font-family: var(--font-display); font-weight: 600; color: var(--ink-dark);">${auth.name}</div>
                    <div style="font-family: var(--font-mono); font-size: 0.85rem; color: var(--ink-fade);">${auth.url}</div>
                </div>
                <span class="status-badge ${auth.status}">${auth.status === 'synced' ? '✓ Synced' : '⏳ Pending'}</span>
            </div>
        `).join('');
    }

    // View Item
    viewItem(id, type) {
        console.log(`Viewing ${type}:`, id);
        // TODO: Implement detailed view modal
    }

    // Edit Item
    editItem(id, type) {
        console.log(`Editing ${type}:`, id);
        // TODO: Implement edit modal
    }

    // Generate commit hash
    generateHash(data) {
        return 'commit-' + Date.now().toString(36) + Math.random().toString(36).substr(2);
    }

    // Commit changes
    async commit(message, author) {
        const commit = {
            hash: this.generateHash(),
            message,
            author,
            timestamp: Date.now(),
            changes: {
                texts: this.data.texts.length,
                entities: this.data.entities.length,
                annotations: this.data.annotations.length
            }
        };

        await this.addToStore('commits', commit);
        this.data.commits.push(commit);
        
        console.log('Committed:', commit);
        return commit;
    }
}

// Modal Management
function showModal(type) {
    const modal = document.getElementById('modal');
    const content = document.getElementById('modalContent');
    
    let modalHTML = '<button class="modal-close" onclick="closeModal()">×</button>';

    switch(type) {
        case 'commit':
            modalHTML += `
                <h2 style="font-family: var(--font-display); font-size: 2rem; margin-bottom: var(--space-md);">Commit Changes</h2>
                <form onsubmit="handleCommit(event)">
                    <div class="form-group">
                        <label class="form-label">Commit Message</label>
                        <input type="text" class="form-input" name="message" placeholder="e.g., Add Byzantine annotations" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Author</label>
                        <input type="text" class="form-input" name="author" placeholder="Your name or ORCID" required>
                    </div>
                    <button type="submit" class="btn primary" style="width: 100%;">Commit</button>
                </form>
            `;
            break;

        case 'clone':
            modalHTML += `
                <h2 style="font-family: var(--font-display); font-size: 2rem; margin-bottom: var(--space-md);">Clone Dataset</h2>
                <form onsubmit="handleClone(event)">
                    <div class="form-group">
                        <label class="form-label">Dataset URN</label>
                        <input type="text" class="form-input" name="urn" placeholder="urn:codex:homeric_core@v1" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Sync Mode</label>
                        <select class="form-select" name="syncMode">
                            <option value="mirror">Mirror (Full Copy)</option>
                            <option value="proxy">Proxy (On-Demand)</option>
                        </select>
                    </div>
                    <button type="submit" class="btn primary" style="width: 100%;">Clone</button>
                </form>
            `;
            break;

        case 'newText':
            modalHTML += `
                <h2 style="font-family: var(--font-display); font-size: 2rem; margin-bottom: var(--space-md);">Add Text Fragment</h2>
                <form onsubmit="handleNewText(event)">
                    <div class="form-group">
                        <label class="form-label">URN</label>
                        <input type="text" class="form-input" name="urn" placeholder="urn:codex:work@source:location" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Content</label>
                        <textarea class="form-textarea" name="content" placeholder="Original text content..." required></textarea>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Language (ISO 639-3)</label>
                        <input type="text" class="form-input" name="language" placeholder="e.g., grc, lat, eng" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Translation (Optional)</label>
                        <textarea class="form-textarea" name="translation" placeholder="English translation..."></textarea>
                    </div>
                    <button type="submit" class="btn primary" style="width: 100%;">Add Text</button>
                </form>
            `;
            break;

        case 'newEntity':
            modalHTML += `
                <h2 style="font-family: var(--font-display); font-size: 2rem; margin-bottom: var(--space-md);">Create Entity</h2>
                <form onsubmit="handleNewEntity(event)">
                    <div class="form-group">
                        <label class="form-label">Entity ID</label>
                        <input type="text" class="form-input" name="id" placeholder="e.g., achilles" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Type</label>
                        <select class="form-select" name="type" required>
                            <option value="Character">Character</option>
                            <option value="Place">Place</option>
                            <option value="Concept">Concept</option>
                            <option value="Event">Event</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Label (English)</label>
                        <input type="text" class="form-input" name="labelEn" placeholder="e.g., Achilles" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Label (Greek) - Optional</label>
                        <input type="text" class="form-input" name="labelGrc" placeholder="e.g., Ἀχιλλεύς">
                    </div>
                    <button type="submit" class="btn primary" style="width: 100%;">Create Entity</button>
                </form>
            `;
            break;

        case 'newAnnotation':
            modalHTML += `
                <h2 style="font-family: var(--font-display); font-size: 2rem; margin-bottom: var(--space-md);">Add Annotation</h2>
                <form onsubmit="handleNewAnnotation(event)">
                    <div class="form-group">
                        <label class="form-label">Text URN</label>
                        <input type="text" class="form-input" name="textUrn" placeholder="urn:codex:..." required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Entity URN</label>
                        <input type="text" class="form-input" name="entityUrn" placeholder="urn:codex:entity/..." required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Character Position (Start)</label>
                        <input type="number" class="form-input" name="start" placeholder="0" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Character Position (End)</label>
                        <input type="number" class="form-input" name="end" placeholder="10" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Role</label>
                        <input type="text" class="form-input" name="role" placeholder="e.g., subject, mentioned" required>
                    </div>
                    <button type="submit" class="btn primary" style="width: 100%;">Add Annotation</button>
                </form>
            `;
            break;

        case 'addRemote':
            modalHTML += `
                <h2 style="font-family: var(--font-display); font-size: 2rem; margin-bottom: var(--space-md);">Add Trusted Authority</h2>
                <form onsubmit="handleAddRemote(event)">
                    <div class="form-group">
                        <label class="form-label">Name</label>
                        <input type="text" class="form-input" name="name" placeholder="e.g., Vatican Library" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">URL</label>
                        <input type="url" class="form-input" name="url" placeholder="https://codex.vatican.va" required>
                    </div>
                    <button type="submit" class="btn primary" style="width: 100%;">Add Remote</button>
                </form>
            `;
            break;
    }

    content.innerHTML = modalHTML;
    modal.classList.add('active');
}

function closeModal() {
    document.getElementById('modal').classList.remove('active');
}

// Form Handlers
async function handleCommit(event) {
    event.preventDefault();
    const formData = new FormData(event.target);
    const message = formData.get('message');
    const author = formData.get('author');
    
    await app.commit(message, author);
    closeModal();
    alert('Changes committed successfully!');
}

function handleClone(event) {
    event.preventDefault();
    const formData = new FormData(event.target);
    console.log('Cloning:', Object.fromEntries(formData));
    closeModal();
    alert('Dataset cloning initiated!');
}

async function handleNewText(event) {
    event.preventDefault();
    const formData = new FormData(event.target);
    
    const text = {
        urn: formData.get('urn'),
        content: formData.get('content'),
        language: formData.get('language'),
        metadata: {
            translation: formData.get('translation') || undefined
        }
    };
    
    await app.addToStore('texts', text);
    app.data.texts.push(text);
    app.renderTexts();
    app.renderRecentActivity();
    closeModal();
    alert('Text added successfully!');
}

async function handleNewEntity(event) {
    event.preventDefault();
    const formData = new FormData(event.target);
    
    const entity = {
        urn: `urn:codex:entity/${formData.get('id')}`,
        type: formData.get('type'),
        labels: {
            en: formData.get('labelEn'),
            grc: formData.get('labelGrc') || undefined
        },
        properties: {}
    };
    
    await app.addToStore('entities', entity);
    app.data.entities.push(entity);
    app.renderEntities();
    app.renderRecentActivity();
    closeModal();
    alert('Entity created successfully!');
}

async function handleNewAnnotation(event) {
    event.preventDefault();
    const formData = new FormData(event.target);
    
    const annotation = {
        textUrn: formData.get('textUrn'),
        entityUrn: formData.get('entityUrn'),
        start: parseInt(formData.get('start')),
        end: parseInt(formData.get('end')),
        role: formData.get('role'),
        certainty: 1.0,
        contributor: 'local-user',
        timestamp: Date.now()
    };
    
    await app.addToStore('annotations', annotation);
    app.data.annotations.push(annotation);
    app.renderAnnotations();
    app.renderRecentActivity();
    closeModal();
    alert('Annotation added successfully!');
}

function handleAddRemote(event) {
    event.preventDefault();
    const formData = new FormData(event.target);
    console.log('Adding remote:', Object.fromEntries(formData));
    closeModal();
    alert('Trusted authority added!');
}

// Initialize App
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new CodexApp();
        // Bind sync button if present
        const btn = document.getElementById('syncVeilBtn');
        if (btn) {
            btn.addEventListener('click', async () => {
                try {
                    btn.disabled = true;
                    btn.textContent = 'Syncing...';
                    await app.syncWithVeil();
                    alert('Sync complete');
                } catch (e) {
                    console.error('sync failed', e);
                    alert('Sync failed: ' + e.message);
                } finally {
                    btn.disabled = false;
                    btn.textContent = 'Sync with Veil';
                }
            });
        }
});

// Listen for messages from host (Veil)
window.addEventListener('message', (ev) => {
    if (!ev.data || !ev.data.type) return;
    if (ev.data.type === 'veil:node-selected') {
        const node = ev.data.node;
        // Attempt to focus the node in the Codex UI
        if (node && node.id) {
            // Assume there's a function findAndOpenNode in the prototype
            if (typeof window.findAndOpenNode === 'function') {
                window.findAndOpenNode(node.id);
            } else {
                console.log('Codex received node-selected for', node);
            }
        }
    }
});

// Wire the Sync with Veil button if present
document.addEventListener('DOMContentLoaded', () => {
    const btn = document.getElementById('syncWithVeilBtn');
    if (btn) btn.addEventListener('click', () => {
        if (typeof app !== 'undefined' && typeof app.syncWithVeil === 'function') {
            app.syncWithVeil().then(() => alert('Sync complete')).catch((err) => alert('Sync failed: ' + err));
        } else {
            alert('Sync not available');
        }
    });
});

// Sync data from Veil Codex endpoints into local IndexedDB
CodexApp.prototype.syncWithVeil = async function() {
    // Fetch commits
    const res = await fetch('/api/codex/commits?limit=100');
    if (!res.ok) throw new Error('Failed to fetch commits');
    const commits = await res.json();
    // For each commit, fetch objects and store
    for (const c of commits) {
        if (!c.objects || !Array.isArray(c.objects)) continue;
        for (const h of c.objects) {
            try {
                const ob = await fetch(`/api/codex/object?hash=${encodeURIComponent(h)}`);
                if (!ob.ok) continue;
                const payload = await ob.json();
                // Simple heuristics: store based on keys
                if (payload.urn && (payload.type === 'Text' || payload.type === 'text' || payload.content)) {
                    await this.addToStore('texts', payload);
                } else if (payload.urn && (payload.type === 'Entity' || payload.type === 'entity' || payload.labels)) {
                    await this.addToStore('entities', payload);
                } else {
                    // fallback to annotations store
                    await this.addToStore('annotations', { id: h, payload });
                }
            } catch (e) {
                console.warn('object fetch/store failed', h, e);
            }
        }
    }
    // reload and render
    await this.loadData();
    this.renderAll();
};
