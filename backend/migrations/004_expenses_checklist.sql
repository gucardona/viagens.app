CREATE TABLE expenses (
    id           INTEGER PRIMARY KEY,
    trip_id      INTEGER NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    item_id      INTEGER REFERENCES items(id) ON DELETE SET NULL,
    title        TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    currency     TEXT NOT NULL,
    category     TEXT NOT NULL,
    paid_by      INTEGER NOT NULL REFERENCES users(id),
    payer_share  INTEGER NOT NULL DEFAULT 50,
    date         TEXT NOT NULL,
    rate         REAL,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX expenses_trip ON expenses(trip_id);

CREATE TABLE rates (
    date     TEXT NOT NULL,
    currency TEXT NOT NULL,
    brl      REAL NOT NULL,
    PRIMARY KEY (date, currency)
);

CREATE TABLE checklist (
    id      INTEGER PRIMARY KEY,
    trip_id INTEGER NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    text    TEXT NOT NULL,
    done    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX checklist_trip ON checklist(trip_id);
