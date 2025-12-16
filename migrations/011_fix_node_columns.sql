-- This migration uses a safe approach to add columns to nodes table
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN
-- So we check and add only if needed

-- The columns will be added one at a time with error suppression in the Go code
