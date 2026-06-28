-- 003_missing_geo.sql — coordinates on missing_persons for near-me search.
ALTER TABLE missing_persons ADD COLUMN IF NOT EXISTS lat double precision;
ALTER TABLE missing_persons ADD COLUMN IF NOT EXISTS lng double precision;
