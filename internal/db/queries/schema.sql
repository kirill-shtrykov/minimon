-- Create "metric" table
CREATE TABLE metric (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL,
    type TEXT NOT NULL,
    value BLOB NOT NULL,
    date TIMESTAMP DEFAULT (datetime('now','localtime'))
);
