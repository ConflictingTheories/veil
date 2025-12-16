-- Node URIs table


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
CREATE TABLE IF NOT EXISTS media_v2 (
    id TEXT PRIMARY KEY,
    filename TEXT,
    storage_url TEXT,
    checksum TEXT UNIQUE,
    mime_type TEXT,
    size INTEGER,
    uploaded_by TEXT,
    created_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_media_v2_checksum ON media_v2(checksum);

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

-- Search hints (columns may not exist yet, ignore errors)
