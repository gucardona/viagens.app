ALTER TABLE trips ADD COLUMN share_token TEXT;
CREATE UNIQUE INDEX trips_share_token ON trips(share_token) WHERE share_token IS NOT NULL;
