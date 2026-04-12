CREATE TABLE IF NOT EXISTS newsletter_sends (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject     VARCHAR(500) NOT NULL,
    body        TEXT NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'pending',
    sent_count  INT NOT NULL DEFAULT 0,
    fail_count  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);