CREATE TABLE IF NOT EXISTS media (
  id TEXT PRIMARY KEY,
  node_id TEXT,
  filename TEXT NOT NULL,
  original_filename TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  file_size INTEGER NOT NULL,
  blob_data BLOB NOT NULL,
  hash TEXT,
  width INTEGER,
  height INTEGER,
  duration INTEGER,
  created_at INTEGER NOT NULL,
  uploaded_by TEXT,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS media_library (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  media_id TEXT NOT NULL,
  category TEXT,
  tags TEXT,
  description TEXT,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (media_id) REFERENCES media(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_media_node_id ON media(node_id);
CREATE INDEX idx_media_hash ON media(hash);
CREATE INDEX idx_media_library_user_id ON media_library(user_id);
