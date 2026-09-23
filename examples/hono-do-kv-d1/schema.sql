-- D1 Database Schema for Hono Notes App
CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  priority INTEGER DEFAULT 1,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Seed initial test record
INSERT OR IGNORE INTO notes (id, title, content, priority)
VALUES ('note_initial', 'Welcome to D1 on Cubit', 'Relational SQLite storage running directly under Cloudflare Workers.', 1);
