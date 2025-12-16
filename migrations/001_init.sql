CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  parent_id TEXT,
  path TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  content TEXT,
  mime_type TEXT,
  created_at INTEGER NOT NULL,
  modified_at INTEGER NOT NULL,
  deleted_at INTEGER
);

CREATE INDEX idx_nodes_path ON nodes(path);
CREATE TABLE IF NOT EXISTS config (key TEXT PRIMARY KEY, value TEXT);
INSERT OR IGNORE INTO config VALUES ('version', '1');
