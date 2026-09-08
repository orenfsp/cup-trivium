ALTER TABLE settings ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

INSERT INTO settings (key, value) VALUES ('expert_active_limit', 5)
ON CONFLICT (key) DO NOTHING;
