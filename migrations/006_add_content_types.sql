CREATE TABLE IF NOT EXISTS blog_posts (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  excerpt TEXT,
  publish_date INTEGER,
  category TEXT,
  allow_comments INTEGER DEFAULT 0,
  view_count INTEGER DEFAULT 0,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS embedded_content (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  embed_url TEXT NOT NULL,
  embed_type TEXT,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS mind_maps (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  title TEXT,
  layout TEXT,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS mind_map_nodes (
  id TEXT PRIMARY KEY,
  mind_map_id TEXT NOT NULL,
  title TEXT NOT NULL,
  node_type TEXT,
  x INTEGER,
  y INTEGER,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (mind_map_id) REFERENCES mind_maps(id)
);

CREATE TABLE IF NOT EXISTS content_blocks (
  id TEXT PRIMARY KEY,
  node_id TEXT,
  block_type TEXT,
  content TEXT,
  order_index INTEGER,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE INDEX idx_blog_posts_node_id ON blog_posts(node_id);
CREATE INDEX idx_blog_posts_slug ON blog_posts(slug);
