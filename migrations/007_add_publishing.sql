CREATE TABLE IF NOT EXISTS exports (
  id TEXT PRIMARY KEY,
  node_id TEXT,
  export_type TEXT,
  export_format TEXT,
  export_path TEXT,
  exported_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS publishing_channels (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  channel_type TEXT,
  config TEXT,
  enabled INTEGER DEFAULT 1,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS publish_history (
  id TEXT PRIMARY KEY,
  node_id TEXT,
  channel_id TEXT,
  published_at INTEGER NOT NULL,
  status TEXT,
  FOREIGN KEY (node_id) REFERENCES nodes(id),
  FOREIGN KEY (channel_id) REFERENCES publishing_channels(id)
);

CREATE TABLE IF NOT EXISTS plugins (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  version TEXT,
  script_path TEXT,
  enabled INTEGER DEFAULT 0,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS plugin_hooks (
  id TEXT PRIMARY KEY,
  plugin_id TEXT NOT NULL,
  hook_name TEXT NOT NULL,
  hook_fn TEXT,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (plugin_id) REFERENCES plugins(id)
);

CREATE INDEX idx_exports_node_id ON exports(node_id);
CREATE INDEX idx_publish_history_node_id ON publish_history(node_id);
