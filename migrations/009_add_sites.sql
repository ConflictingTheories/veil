-- Sites/Projects Table
CREATE TABLE IF NOT EXISTS sites (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT DEFAULT 'project',
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL
);

-- Add site_id to nodes (ignore if column already exists)
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS site_id TEXT REFERENCES sites(id);

CREATE INDEX IF NOT EXISTS idx_nodes_site_id ON nodes(site_id);
CREATE INDEX IF NOT EXISTS idx_sites_name ON sites(name);
