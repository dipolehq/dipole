ALTER TABLE log_streams ADD COLUMN IF NOT EXISTS last_exported_at timestamp with time zone;
