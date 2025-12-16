-- Extend nodes with canonical URI, slug, JSON body & metadata
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS slug TEXT;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS canonical_uri TEXT;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS body TEXT; -- JSON structured body
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS metadata TEXT; -- JSON metadata
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'draft';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS visibility TEXT DEFAULT 'public';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS site_id TEXT REFERENCES sites(id);

CREATE TABLE IF NOT EXISTS node_uris (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    uri TEXT NOT NULL UNIQUE,
    is_primary INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_node_uris_node ON node_uris(node_id);

-- Plugins registry for plugin manifests and state
CREATE TABLE IF NOT EXISTS plugins_registry (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE,
    manifest TEXT,
    enabled INTEGER DEFAULT 0,
    created_at INTEGER,
    updated_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_plugins_slug ON plugins_registry(slug);

-- Media table with checksums and storage metadata
CREATE TABLE IF NOT EXISTS media (
    id TEXT PRIMARY KEY,
    filename TEXT,
    storage_url TEXT,
    checksum TEXT UNIQUE,
    mime_type TEXT,
    size INTEGER,
    uploaded_by TEXT,
    created_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_media_checksum ON media(checksum);

-- Tags and node-tags join for faster lookups
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE,
    color TEXT
);

CREATE TABLE IF NOT EXISTS node_tags (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id),
    UNIQUE(node_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_node_tags_node ON node_tags(node_id);

-- Simple search hints
CREATE INDEX IF NOT EXISTS idx_nodes_slug ON nodes(slug);
CREATE INDEX IF NOT EXISTS idx_nodes_canonical_uri ON nodes(canonical_uri);
