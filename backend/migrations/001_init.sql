CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL
);

CREATE TABLE trips (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    destination TEXT NOT NULL DEFAULT '',
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE items (
    id                INTEGER PRIMARY KEY,
    trip_id           INTEGER NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    kind              TEXT NOT NULL,
    title             TEXT NOT NULL,
    start_date        TEXT NOT NULL DEFAULT '',
    start_time        TEXT NOT NULL DEFAULT '',
    end_date          TEXT NOT NULL DEFAULT '',
    end_time          TEXT NOT NULL DEFAULT '',
    address           TEXT NOT NULL DEFAULT '',
    lat               REAL,
    lng               REAL,
    confirmation_code TEXT NOT NULL DEFAULT '',
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at        TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX items_trip ON items(trip_id);

CREATE TABLE attachments (
    id           INTEGER PRIMARY KEY,
    trip_id      INTEGER NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    item_id      INTEGER REFERENCES items(id) ON DELETE CASCADE,
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size         INTEGER NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX attachments_trip ON attachments(trip_id);
