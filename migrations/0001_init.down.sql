DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP TABLE IF EXISTS "user";
DROP TABLE IF EXISTS refresh_token;
DROP TABLE IF EXISTS project;