-- タスクに画像URL列を追加

ALTER TABLE tasks ADD COLUMN image_url TEXT DEFAULT NULL;

-- インデックス追加（画像付きタスクの検索用）
CREATE INDEX IF NOT EXISTS idx_tasks_image_url ON tasks(image_url) WHERE image_url IS NOT NULL;

