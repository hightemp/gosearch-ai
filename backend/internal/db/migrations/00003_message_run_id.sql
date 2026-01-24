-- +goose Up
ALTER TABLE messages ADD COLUMN run_id uuid REFERENCES runs(id) ON DELETE SET NULL;
CREATE INDEX messages_run_id_idx ON messages(run_id);

-- +goose Down
DROP INDEX IF EXISTS messages_run_id_idx;
ALTER TABLE messages DROP COLUMN IF EXISTS run_id;
