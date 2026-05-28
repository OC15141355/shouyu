-- migrations/001_initial.sql
CREATE TABLE notes (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  body          TEXT NOT NULL,
  author        TEXT NOT NULL,
  created_at    INTEGER NOT NULL,
  deleted_at    INTEGER NULL
);
CREATE INDEX idx_notes_created ON notes (created_at DESC) WHERE deleted_at IS NULL;
