CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS "user" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS refresh_token (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_token(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_token(expires_at);

CREATE TABLE IF NOT EXISTS project (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_projects_user_id ON project(user_id);

CREATE TABLE task (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('CREATED', 'WORKING_ON', 'CLOSED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, project_id, name)
);

CREATE TABLE IF NOT EXISTS time_record (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    work_timezone TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NULL,
    total_minutes INTEGER NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_time_record_time_order
        CHECK (ended_at IS NULL OR ended_at > started_at),

    CONSTRAINT chk_time_record_total_minutes_non_negative
        CHECK (total_minutes IS NULL OR total_minutes >= 0),

    CONSTRAINT chk_time_record_open_or_closed_consistency
        CHECK (
            (ended_at IS NULL AND total_minutes IS NULL)
            OR
            (ended_at IS NOT NULL AND total_minutes IS NOT NULL)
        )
);

CREATE INDEX IF NOT EXISTS idx_time_record_report
    ON time_record (user_id, project_id, work_date, task_id);

CREATE INDEX IF NOT EXISTS idx_time_record_task_id
    ON time_record (task_id);

CREATE INDEX IF NOT EXISTS idx_time_record_user_id
    ON time_record (user_id);

CREATE INDEX IF NOT EXISTS idx_time_record_project_id
    ON time_record (project_id);

CREATE INDEX IF NOT EXISTS idx_time_record_work_date
    ON time_record (work_date);

-- Only one active timer per user
CREATE UNIQUE INDEX IF NOT EXISTS uq_time_record_one_active_per_user
    ON time_record (user_id)
    WHERE ended_at IS NULL;