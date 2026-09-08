
CREATE TABLE IF NOT EXISTS appeal_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id  UUID NOT NULL REFERENCES appeals(id) ON DELETE CASCADE,
    actor_id   UUID,
    actor_role TEXT NOT NULL,
    event_type TEXT NOT NULL,
    old_value  TEXT,
    new_value  TEXT,
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS appeal_events_appeal_idx ON appeal_events (appeal_id, created_at);

CREATE TABLE IF NOT EXISTS feedback (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id  UUID NOT NULL UNIQUE REFERENCES appeals(id) ON DELETE CASCADE,
    helped     BOOLEAN,
    rating     INT CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS crisis_contacts (
    appeal_id  UUID PRIMARY KEY REFERENCES appeals(id) ON DELETE CASCADE,
    contact    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS complaints (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id  UUID NOT NULL REFERENCES appeals(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
