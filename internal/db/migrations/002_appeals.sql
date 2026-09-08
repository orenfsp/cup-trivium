
CREATE TABLE IF NOT EXISTS appeals (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_hash         TEXT NOT NULL UNIQUE,
    applicant_type     TEXT NOT NULL CHECK (applicant_type IN ('schoolchild', 'parent', 'teacher')),
    category_id        UUID REFERENCES categories(id),
    free_text_mode     BOOLEAN NOT NULL DEFAULT FALSE,
    description        TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'assigned', 'in_progress', 'needs_clarification',
                          'answer_ready', 'completed', 'returned', 'rejected', 'closed_no_response')),
    priority           TEXT NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('low', 'normal', 'urgent')),
    crisis_detected    BOOLEAN NOT NULL DEFAULT FALSE,
    assigned_expert_id UUID REFERENCES users(id),
    rejection_reason   TEXT,
    recommendation     TEXT,
    return_count       INT NOT NULL DEFAULT 0,
    transfer_requested BOOLEAN NOT NULL DEFAULT FALSE,
    version            INT NOT NULL DEFAULT 1,
    idempotency_key    TEXT UNIQUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS appeals_queue_idx ON appeals (status, crisis_detected DESC, created_at);

CREATE TABLE IF NOT EXISTS intake_answers (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id UUID NOT NULL REFERENCES appeals(id) ON DELETE CASCADE,
    question  TEXT NOT NULL,
    answer    TEXT NOT NULL
);


CREATE TABLE IF NOT EXISTS messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id       UUID NOT NULL REFERENCES appeals(id) ON DELETE CASCADE,
    author_type     TEXT NOT NULL CHECK (author_type IN ('applicant', 'expert', 'operator')),
    author_id       UUID,
    text            TEXT NOT NULL,
    idempotency_key TEXT UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS messages_appeal_idx ON messages (appeal_id, created_at);


CREATE TABLE IF NOT EXISTS internal_notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id  UUID NOT NULL REFERENCES appeals(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users(id),
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS internal_notes_appeal_idx ON internal_notes (appeal_id, created_at);

CREATE TABLE IF NOT EXISTS attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id    UUID NOT NULL REFERENCES appeals(id) ON DELETE CASCADE,
    storage_name TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    size         INT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS attachments_appeal_idx ON attachments (appeal_id);

CREATE TABLE IF NOT EXISTS appeal_participants (
    appeal_id        UUID NOT NULL REFERENCES appeals(id) ON DELETE CASCADE,
    expert_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant_role TEXT NOT NULL CHECK (participant_role IN ('responsible', 'contributor')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (appeal_id, expert_id)
);


CREATE UNIQUE INDEX IF NOT EXISTS appeal_one_responsible_expert
    ON appeal_participants (appeal_id)
    WHERE participant_role = 'responsible';
