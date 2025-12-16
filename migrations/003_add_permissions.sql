CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  email TEXT UNIQUE,
  password_hash TEXT,
  role TEXT DEFAULT 'viewer',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  is_active INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS permissions (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  user_id TEXT,
  role TEXT NOT NULL,
  granted_at INTEGER NOT NULL,
  granted_by TEXT,
  FOREIGN KEY (node_id) REFERENCES nodes(id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  UNIQUE(node_id, user_id)
);

CREATE TABLE IF NOT EXISTS node_visibility (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL UNIQUE,
  visibility TEXT DEFAULT 'private',
  share_token TEXT UNIQUE,
  expires_at INTEGER,
  allow_comments INTEGER DEFAULT 0,
  allow_collaboration INTEGER DEFAULT 0,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE INDEX idx_permissions_node_id ON permissions(node_id);
CREATE INDEX idx_permissions_user_id ON permissions(user_id);
CREATE INDEX idx_node_visibility_token ON node_visibility(share_token);
