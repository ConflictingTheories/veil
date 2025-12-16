CREATE TABLE IF NOT EXISTS versions (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  version_number INTEGER NOT NULL,
  content TEXT NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  published_at INTEGER,
  published_by TEXT,
  created_at INTEGER NOT NULL,
  modified_at INTEGER NOT NULL,
  is_current INTEGER DEFAULT 0,
  FOREIGN KEY (node_id) REFERENCES nodes(id),
  UNIQUE(node_id, version_number)
);

CREATE TABLE IF NOT EXISTS snapshots (
  id TEXT PRIMARY KEY,
  timestamp INTEGER NOT NULL,
  description TEXT,
  node_count INTEGER,
  created_by TEXT,
  created_at INTEGER NOT NULL,
  is_rollback_point INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS snapshot_nodes (
  id TEXT PRIMARY KEY,
  snapshot_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  content TEXT NOT NULL,
  title TEXT NOT NULL,
  metadata TEXT,
  FOREIGN KEY (snapshot_id) REFERENCES snapshots(id),
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE INDEX idx_versions_node_id ON versions(node_id);
CREATE INDEX idx_versions_status ON versions(status);
