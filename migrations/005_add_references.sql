CREATE TABLE IF NOT EXISTS node_references (
  id TEXT PRIMARY KEY,
  source_node_id TEXT NOT NULL,
  target_node_id TEXT NOT NULL,
  link_type TEXT NOT NULL,
  link_text TEXT,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (source_node_id) REFERENCES nodes(id),
  FOREIGN KEY (target_node_id) REFERENCES nodes(id),
  UNIQUE(source_node_id, target_node_id, link_type)
);

CREATE TABLE IF NOT EXISTS tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  color TEXT,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS node_tags (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  tag_id TEXT NOT NULL,
  added_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id),
  UNIQUE(node_id, tag_id)
);

CREATE TABLE IF NOT EXISTS citations (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  citation_key TEXT NOT NULL,
  authors TEXT,
  title TEXT,
  year INTEGER,
  publisher TEXT,
  url TEXT,
  doi TEXT,
  citation_format TEXT DEFAULT 'APA',
  raw_bibtex TEXT,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id),
  UNIQUE(node_id, citation_key)
);

CREATE INDEX idx_node_references_source ON node_references(source_node_id);
CREATE INDEX idx_node_references_target ON node_references(target_node_id);
