-- 初期スキーマ: ユーザー、タスク、認証コード

-- ユーザーテーブル
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- メールアドレスインデックス（高速検索用）
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_verified ON users(is_verified);

-- タスクテーブル
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL,
    due_date TIMESTAMP,
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP,
    priority TEXT DEFAULT 'medium',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ユーザーIDとタスクインデックス（高速検索用）
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(is_completed);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

-- タスクタグテーブル（多対多リレーション）
CREATE TABLE IF NOT EXISTS task_tags (
    task_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (task_id, tag),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- 認証コードテーブル
CREATE TABLE IF NOT EXISTS verification_codes (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    code TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    used BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 認証コードインデックス（高速検索用）
CREATE INDEX IF NOT EXISTS idx_verification_user_id ON verification_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_verification_code ON verification_codes(code);
CREATE INDEX IF NOT EXISTS idx_verification_expires ON verification_codes(expires_at);

-- レート制限テーブル（オプション: メモリベースでも可）
CREATE TABLE IF NOT EXISTS rate_limits (
    key TEXT PRIMARY KEY,
    tokens DOUBLE PRECISION NOT NULL,
    last_update TIMESTAMP NOT NULL
);
