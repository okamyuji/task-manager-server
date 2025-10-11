# Task Manager Server

Golang REST APIによるタスク管理サーバー

## 目次

- [プロジェクト構成](#プロジェクト構成)
- [技術スタック](#技術スタック)
- [アーキテクチャ](#アーキテクチャ)
- [クイックスタート](#クイックスタート)
- [API仕様](#api仕様)
- [セキュリティ](#セキュリティ)
- [環境変数](#環境変数)
- [本番環境への展開](#本番環境への展開)
- [トラブルシューティング](#トラブルシューティング)

---

## プロジェクト構成

```shell
task-manager-server/
├── main.go                    # エントリーポイント、依存性注入
├── handlers/
│   ├── auth_handler.go       # 認証ハンドラー
│   ├── task_handler.go       # タスクハンドラー
│   ├── upload_handler.go     # 画像アップロードハンドラー
│   ├── verification_handler.go # メール認証ハンドラー
│   └── response.go           # レスポンスヘルパー
├── database/
│   ├── db.go                 # データベース接続（PostgreSQL）
│   ├── migrations/           # SQLマイグレーション
│   └── repository/           # データアクセス層
│       ├── user_repository.go
│       ├── task_repository.go
│       └── verification_repository.go
├── middleware/
│   ├── middleware.go         # 認証・CORS・ロギング
│   └── rate_limiter.go       # レート制限
├── models/
│   └── models.go             # データモデル
├── storage/
│   └── storage.go            # ストレージサービス（ローカル/S3）
├── utils/
│   ├── jwt.go                # JWT生成・検証
│   └── password.go           # パスワードハッシュ化
├── validation/
│   └── validator.go          # バリデーション
├── email/
│   └── email_service.go      # メール送信（MailHog/Resend）
├── uploads/                  # アップロード画像保存先（ローカルのみ）
├── compose.yaml              # Docker Compose設定
├── Dockerfile                # Dockerイメージ定義
└── env.example               # 環境変数テンプレート
```

---

## 技術スタック

- **Go**: 1.25.0
- **標準ライブラリ**: net/http, encoding/json, log/slog
- **データベース**: PostgreSQL 16
- **ストレージ**: ローカル（開発）/ S3互換（本番: Leapcell Object Storage）
- **外部依存**:
  - github.com/google/uuid
  - github.com/lib/pq (PostgreSQL driver)
  - golang.org/x/crypto/bcrypt
  - github.com/aws/aws-sdk-go-v2 (S3 SDK)
  - github.com/resend/resend-go/v2 (Resend SDK)
- **認証**: JWT (HMAC-SHA256)
- **メール**: MailHog (開発環境) / Resend API (本番環境)

---

## アーキテクチャ

### レイヤードアーキテクチャ

本システムは、責務を明確に分離したレイヤードアーキテクチャを採用しています。

```text
┌─────────────────────────────────────────────────────────────────┐
│                      Go REST API Server                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                   HTTP Router (mux)                      │   │
│  └──────────────────────────────────────────────────────────┘   │
│       │                    │                    │                │
│       ▼                    ▼                    ▼                │
│  ┌─────────┐         ┌─────────┐         ┌─────────┐           │
│  │ Handler │         │ Handler │         │ Handler │           │
│  │  Layer  │         │  Layer  │         │  Layer  │           │
│  └─────────┘         └─────────┘         └─────────┘           │
│       │                    │                    │                │
│       ▼                    ▼                    ▼                │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                   Middleware Layer                       │   │
│  │         Auth / CORS / Logging / Rate Limiting           │   │
│  └──────────────────────────────────────────────────────────┘   │
│       │                    │                    │                │
│       ▼                    ▼                    ▼                │
│  ┌─────────┐   ┌──────────┐   ┌─────────┐   ┌─────────┐       │
│  │  Utils  │   │   Repo   │   │  Email  │   │ Storage │       │
│  │  Layer  │   │   Layer  │   │ Service │   │ Service │       │
│  └─────────┘   └──────────┘   └─────────┘   └─────────┘       │
│                      │              │              │             │
│                      ▼              ▼              ▼             │
│              ┌──────────────┐  ┌────────┐  ┌──────────┐        │
│              │  PostgreSQL  │  │MailHog/ │  │  Local/  │        │
│              │   Database   │  │Resend  │  │   S3     │        │
│              └──────────────┘  └────────┘  └──────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

### 各層の責務

| レイヤー | ディレクトリ | 責務 |
|---------|------------|-----|
| **Handlers** | `handlers/` | ビジネスロジック、HTTPリクエスト/レスポンス処理 |
| **Middleware** | `middleware/` | 横断的関心事（認証・CORS・ロギング・レート制限） |
| **Models** | `models/` | APIモデル、リクエスト/レスポンス型 |
| **Repository** | `database/repository/` | データアクセス抽象化（PostgreSQL） |
| **Storage** | `storage/` | ファイルストレージ抽象化（ローカル/S3） |
| **Utils** | `utils/` | JWT生成/検証、パスワードハッシュ化 |
| **Validation** | `validation/` | 入力検証 |
| **Email** | `email/` | メール送信サービス（MailHog/Resend） |

### JWT認証フロー

```text
1. クライアント → POST /auth/register (email, password, name)
2. サーバー → ユーザー作成（未認証状態）+ 認証コード送信
3. クライアント → POST /auth/verify (email, code)
4. サーバー → ユーザー認証完了
5. クライアント → POST /auth/login (email, password)
6. サーバー → JWTトークン生成 (HMAC-SHA256)
7. サーバー → クライアント (accessToken: 15分, refreshToken: 7日)
8. クライアント → 以降のリクエストに Authorization: Bearer {token}
9. トークン期限切れ → POST /auth/refresh で更新
```

---

## クイックスタート

### オプションA: Docker Compose（推奨）

#### 前提条件

- Docker Engine 20.10以上
- Docker Compose v2.0以上

#### 起動手順

```bash
# 1. 環境変数ファイルを作成
cp env.example .env

# 2. .envファイルを編集
# 必要に応じて以下の値を変更：
# - JWT_SECRET: 強力なランダム文字列に変更（本番環境）
# - ALLOWED_ORIGINS: 許可するドメインを指定（開発環境では*でOK）
# - RESEND_API_KEY: 本番環境でメール送信する場合に設定

# 3. Dockerコンテナをビルド＆起動
docker compose up -d --build

# 4. ログを確認
docker compose logs -f api

# 5. MailHog WebUIにアクセス
open http://localhost:8025
```

**重要**: `.env`ファイルは`.gitignore`に含まれており、Gitにコミットされません。本番環境では必ず独自の値を設定してください。

#### サービスURL

- **APIサーバー**: `http://localhost:8080`
- **PostgreSQL**: `localhost:5432`（ユーザー: `taskmanager`, DB: `taskmanager`）
- **MailHog WebUI**: `http://localhost:8025`（メール確認用）

#### コンテナ管理

```bash
# 停止（データは保持）
docker compose stop

# 停止 + コンテナ削除（PostgreSQLデータは保持）
docker compose down

# 停止 + 全データ削除（PostgreSQLデータも削除）
docker compose down -v
```

#### 環境変数の設定

Docker Composeは`.env`ファイルから自動的に環境変数を読み込みます。

**開発環境の.env例:**

```env
# JWT秘密鍵（開発環境用）
JWT_SECRET=dev-secret-key-change-in-production

# CORS設定（開発環境では全て許可）
ALLOWED_ORIGINS=*

# データベース（Docker Composeで自動設定）
# DATABASE_URL=postgresql://taskmanager:taskmanager_dev_password@postgres:5432/taskmanager?sslmode=disable

# ストレージ（開発環境: ローカル）
STORAGE_TYPE=local
BASE_URL=http://localhost:8080

# メール送信（開発環境: MailHog）
MAIL_FROM=noreply@taskmanager.local
# RESEND_API_KEYは設定しない（MailHogを使用）

# デバッグモード
DEBUG=true
```

**本番環境の.env例（Leapcell）:**

```env
# JWT秘密鍵（本番環境: 強力なランダム文字列）
JWT_SECRET=xK9$mP2@vN7#qR4&wT8!yU3%aS6^dF1*gH5+jL0-

# CORS設定（本番環境: 許可するドメインのみ）
ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# データベース（Leapcell PostgreSQL）
DATABASE_URL=postgresql://user:pass@your-db.leapcell.io:5432/taskmanager?sslmode=require

# ストレージ（本番環境: Leapcell Object Storage）
STORAGE_TYPE=s3
S3_ENDPOINT=https://objstorage.leapcell.io
S3_REGION=us-east-1
S3_BUCKET=task-manager-images
S3_ACCESS_KEY=XXXXXXX
S3_SECRET_KEY=XXXXXXX
S3_CDN_URL=https://random.leapcellobj.com/task-manager-images-random
BASE_URL=https://your-app.leapcell.dev

# メール送信（本番環境: Resend）
RESEND_API_KEY=re_xxxxxxxxxxxxx
MAIL_FROM=noreply@yourdomain.com

# デバッグモードOFF
DEBUG=false
```

**設定の優先順位（高→低）:**

1. `compose.yaml`の`environment`セクション（最優先）
2. `env_file`で読み込まれた`.env`の値
3. `${変数:-デフォルト}`形式のデフォルト値

**環境変数一覧:**

| 変数名 | 説明 | デフォルト値 | 開発環境 | 本番環境 |
|--------|------|-------------|---------|---------|
| `JWT_SECRET` | JWT秘密鍵 | `default-secret-...` | 任意 | **必須** |
| `ALLOWED_ORIGINS` | CORS許可オリジン | `*` | `*` | **必須** |
| `DATABASE_URL` | PostgreSQL接続URL | `postgresql://...` | compose.yamlで設定 | **必須** |
| `STORAGE_TYPE` | ストレージタイプ | `local` | `local` | `s3` |
| `BASE_URL` | ベースURL | `http://localhost:8080` | デフォルト | **必須** |
| `S3_ENDPOINT` | S3エンドポイント | - | 不要 | **必須（S3使用時）** |
| `S3_REGION` | S3リージョン | - | 不要 | **必須（S3使用時）** |
| `S3_BUCKET` | S3バケット名 | - | 不要 | **必須（S3使用時）** |
| `S3_ACCESS_KEY` | S3アクセスキー | - | 不要 | **必須（S3使用時）** |
| `S3_SECRET_KEY` | S3シークレットキー | - | 不要 | **必須（S3使用時）** |
| `S3_CDN_URL` | S3 CDN URL | - | 不要 | 推奨（S3使用時） |
| `RESEND_API_KEY` | Resend APIキー | - | 不要 | **必須** |
| `MAIL_FROM` | 送信元アドレス | `noreply@taskmanager.local` | デフォルト | **必須** |
| `SMTP_HOST` | SMTPホスト | `mailhog` | `mailhog` | 不要 |
| `SMTP_PORT` | SMTPポート | `1025` | `1025` | 不要 |
| `PORT` | サーバーポート | `8080` | - | - |
| `DEBUG` | デバッグモード | `false` | `true` | `false` |

#### データの永続化

以下にデータが保存されます（開発環境）：

- **PostgreSQL**: Dockerボリューム `postgres_data`（永続化）
- **アップロード画像**: `./uploads/`（ローカルディレクトリ）

本番環境（Leapcell）：

- **PostgreSQL**: Leapcellのマネージドデータベース
- **アップロード画像**: Leapcell Object Storage（S3互換）

### オプションB: ローカル実行

```bash
# 依存関係インストール
go mod tidy

# アップロードディレクトリ作成
mkdir -p uploads

# 環境変数設定（必須）
export DATABASE_URL="postgresql://taskmanager:taskmanager_dev_password@localhost:5432/taskmanager?sslmode=disable"
export JWT_SECRET="your-secret-key"
export ALLOWED_ORIGINS="*"
export STORAGE_TYPE="local"
export DEBUG="true"

# サーバー起動（ポート8080）
go run .
```

**注意**:

- ローカル実行時は別途PostgreSQLサーバーが必要です（Docker Composeを推奨）
- メール認証機能はMailHogまたはResendの設定が必要です

---

## API仕様

### 認証エンドポイント

#### ユーザー登録

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "StrongPass123",
  "name": "ユーザー名"
}
```

レスポンス（未認証状態）:

```json
{
  "message": "User created. Please check your email for verification code.",
  "userId": "uuid",
  "email": "user@example.com"
}
```

#### メール認証

```http
POST /auth/verify
Content-Type: application/json

{
  "email": "user@example.com",
  "code": "123456"
}
```

レスポンス:

```json
{
  "message": "Verification successful",
  "userId": "uuid"
}
```

#### 認証コード再送信

```http
POST /auth/resend-code
Content-Type: application/json

{
  "email": "user@example.com"
}
```

#### ログイン

```http
POST /auth/login
Content-Type: application/json

{
  "email": "test@example.com",
  "password": "password123"
}
```

レスポンス:

```json
{
  "accessToken": "eyJhbGc...",
  "refreshToken": "eyJhbGc...",
  "userId": "uuid"
}
```

#### トークンリフレッシュ

```http
POST /auth/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGc..."
}
```

### タスクエンドポイント（要認証）

すべてのタスクエンドポイントには`Authorization`ヘッダーが必要です

```http
Authorization: Bearer {accessToken}
```

| メソッド | エンドポイント | 説明 |
|---------|--------------|------|
| GET | `/tasks` | タスク一覧取得 |
| POST | `/tasks` | タスク作成 |
| GET | `/tasks/{id}` | タスク詳細取得 |
| PUT | `/tasks/{id}` | タスク更新 |
| DELETE | `/tasks/{id}` | タスク削除 |
| PATCH | `/tasks/{id}/complete` | タスク完了 |
| PATCH | `/tasks/{id}/incomplete` | タスク未完了 |

#### タスク作成例

```http
POST /tasks
Authorization: Bearer {accessToken}
Content-Type: application/json

{
  "title": "タスク名",
  "description": "説明",
  "dueDate": "2025-12-31T23:59:59Z",
  "priority": "high",
  "tags": ["仕事", "重要"]
}
```

### 画像アップロード

```http
POST /upload
Authorization: Bearer {accessToken}
Content-Type: multipart/form-data

image: (binary file, 最大10MB)
```

レスポンス:

```json
{
  "url": "http://localhost:8080/uploads/1234567890_uuid.jpg"
}
```

---

## セキュリティ

### 実装済みセキュリティ機能 ✅

1. ✅ **パスワードのハッシュ化（bcrypt）**
   - bcrypt（コスト係数12）によるパスワードハッシュ化
   - `golang.org/x/crypto/bcrypt`使用

2. ✅ **JWT秘密鍵の環境変数化**
   - `JWT_SECRET`環境変数から読み込み
   - デフォルト値は開発環境用のみ
   - `.env`ファイルは`.gitignore`で除外

3. ✅ **レート制限の実装**
   - 認証エンドポイント: 5 req/min per IP
   - タスクAPI: 60 req/min per user
   - アップロード: 10 req/min per user
   - Token Bucketアルゴリズム使用

4. ✅ **PostgreSQLデータベース**
   - トランザクション対応
   - Prepared Statement（SQLインジェクション対策）
   - マイグレーション管理
   - 接続プール最適化

5. ✅ **入力バリデーション強化**
   - メールフォーマット検証（正規表現）
   - パスワード強度検証（8文字以上、英数字含む）
   - タスクデータバリデーション

6. ✅ **本番対応CORS設定**
   - `ALLOWED_ORIGINS`環境変数によるホワイトリスト
   - `Access-Control-Allow-Credentials`対応
   - 開発環境では`*`許可

7. ✅ **メール認証システム**
   - ユーザー登録時の6桁認証コード送信
   - 認証コード有効期限（15分）
   - 認証コード再送信機能

8. ✅ **構造化ログ（slog）**
   - JSON形式のログ出力
   - ログレベル制御（INFO/DEBUG/WARN/ERROR）
   - リクエスト追跡

9. ✅ **JWT認証とユーザー認可**
   - アクセストークン（15分）
   - リフレッシュトークン（7日）
   - ユーザーごとのリソースアクセス制御

### セキュリティのベストプラクティス

#### .envファイルの保護

1. **Gitにコミットしない**

   ```bash
   # .gitignoreに.envが含まれていることを確認
   cat .gitignore | grep .env
   
   # git statusで.envが表示されないことを確認
   git status
   ```

2. **強力なJWT_SECRETを使用**

   ```bash
   # 256ビット（32バイト）のランダム文字列を生成
   openssl rand -base64 32
   ```

3. **本番環境では環境変数を暗号化**
   - クラウドプロバイダーのシークレット管理を使用
   - 例: AWS Secrets Manager, GCP Secret Manager, Azure Key Vault

#### 環境変数のローテーション

定期的に（3-6ヶ月ごと）に`JWT_SECRET`と`RESEND_API_KEY`をローテーションしてください：

```bash
# 1. 新しいJWT_SECRETを生成
openssl rand -base64 32

# 2. .envを更新
# JWT_SECRET=新しい値

# 3. コンテナ再起動
docker compose restart api

# 4. 全ユーザーが再ログインする必要があります
```

### リソースアクセス制御

ユーザーは自分のタスクのみにアクセス可能です：

```go
// 所有者チェックの例
if task.UserID != userID {
    logger.Warn("アクセス権限がありません",
        "task_id", taskID,
        "task_owner", task.UserID,
        "requesting_user", userID,
    )
    respondWithError(w, http.StatusForbidden, "Access denied")
    return
}
```

### 本番環境で必要な追加対応 ⚠️

1. ❗ **HTTPSの使用**
   - Nginx/Caddy でリバースプロキシ
   - Let's Encrypt による証明書取得

2. ✅ **本番用メール送信（Resend対応済み）**
   - 開発環境: MailHog（自動）
   - 本番環境: Resend API（環境変数で自動切り替え）
   - 無料枠: 100通/日（3,000通/月）

3. ❗ **定期的なバックアップ**
   - SQLiteデータベースの定期バックアップ
   - アップロード画像のバックアップ

---

## 環境変数

### 必須環境変数（本番環境）

```env
# JWT秘密鍵（256ビット推奨）
JWT_SECRET=your-super-secret-production-key-here

# 許可するオリジン（カンマ区切り）
ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
```

### メール送信設定

#### 開発環境（MailHog）

```env
# MailHogを使用（デフォルト）
SMTP_HOST=mailhog
SMTP_PORT=1025
MAIL_FROM=noreply@taskmanager.local
```

#### 本番環境（Resend）

```env
# Resend APIキー（https://resend.com/api-keys で取得）
RESEND_API_KEY=re_xxxxxxxxxxxxx

# 送信元メールアドレス（Resendでドメイン認証済み）
MAIL_FROM=noreply@yourdomain.com
```

**Resendの設定手順:**

1. [Resend](https://resend.com/)でアカウント作成
2. ドメイン認証を完了（DNS設定が必要）
3. API Keysページで新しいキーを作成
4. 環境変数`RESEND_API_KEY`に設定

**自動切り替え:**

- `RESEND_API_KEY`が設定されている → Resend API使用（本番環境）
- `RESEND_API_KEY`が未設定 → MailHog使用（開発環境）

### オプション環境変数

```env
# サーバーポート（デフォルト: 8080）
PORT=8080

# データベースパス（デフォルト: ./data/tasks.db）
DATABASE_PATH=./data/tasks.db

# デバッグモード（デフォルト: false）
DEBUG=true
```

---

## 本番環境への展開

### オプションA: Leapcellへのデプロイ（推奨）

Leapcellは、GitHubリポジトリから直接デプロイできるクラウドプラットフォームです。

#### 前提条件

- GitHubアカウント
- [Leapcellアカウント](https://leapcell.io/)
- 本プロジェクトがGitHubにプッシュ済み
- Resendアカウント（メール送信用）
- JWT秘密鍵の準備（下記参照）

#### デプロイ手順

- 環境変数の準備

まず、必要な環境変数を準備します：

```bash
# JWT秘密鍵を生成
openssl rand -base64 32
# 例: your-generated-jwt-secret-key-at-least-32-bytes

# Resend APIキーを取得
# https://resend.com/api-keys で発行
# 例: re_xxxxxxxxxxxxx
```

必須環境変数：

| 変数名 | 説明 | 例 |
|--------|------|-----|
| `JWT_SECRET` | JWT秘密鍵 | `your-jwt-secret...` |
| `DATABASE_URL` | PostgreSQL接続URL | `postgresql://user:pass@host:5432/db` |
| `RESEND_API_KEY` | Resend APIキー | `re_xxxxx...` |
| `MAIL_FROM` | 送信元メールアドレス | `noreply@yourdomain.com` |
| `ALLOWED_ORIGINS` | CORS許可オリジン | `https://yourdomain.com` |
| `STORAGE_TYPE` | ストレージタイプ | `s3` |
| `S3_ENDPOINT` | S3エンドポイント | `https://objstorage.leapcell.io` |
| `S3_REGION` | S3リージョン | `us-east-1` |
| `S3_BUCKET` | S3バケット名 | `task-manager-images-...` |
| `S3_ACCESS_KEY` | S3アクセスキー | `your-access-key...` |
| `S3_SECRET_KEY` | S3シークレットキー | `your-secret-key...` |
| `S3_CDN_URL` | S3 CDN URL | `https://1xg7ah.leapcellobj.com/...` |
| `BASE_URL` | ベースURL | `https://your-app.leapcell.dev` |
| `DEBUG` | デバッグモード | `false` |

- GitHubにプッシュ

```bash
# ローカルの変更をコミット
git add .
git commit -m "準備完了: Leapcellデプロイ用"

# GitHubにプッシュ
git push origin main
```

**重要**: `.env`ファイルは`.gitignore`で除外されているため、Gitにコミットされません。

- PostgreSQLデータベースを作成

1. [https://leapcell.io/](https://leapcell.io/)にログイン
2. 「Databases」→「Create Database」をクリック
3. 「PostgreSQL」を選択
4. データベース名: `taskmanager`
5. リージョンを選択（例: `us-east-1`）
6. 作成完了後、接続URLをコピー（後で使用）

- Object Storageを作成

1. 「Object Storage」→「Create Bucket」をクリック
2. バケット名を入力（例: `task-manager-images-xxx`）
3. リージョンを選択（例: `us-east-1`）
4. 作成完了後、以下の情報をコピー：
   - エンドポイント
   - アクセスキーID
   - シークレットアクセスキー
   - CDN URL

- Leapcellでサービスを作成

1. 「Services」→「New Service」をクリック
2. 「Connect GitHub」→ リポジトリを選択（例: `your-username/task-manager-server`）
3. ブランチを選択（通常は`main`または`master`）
4. Leapcellが`Dockerfile`を自動検出

- デプロイモードの選択

Leapcellには2つのモードがあります：

| モード | 推奨 | 理由 |
|--------|------|------|
| **Serverless** | ✅ 推奨 | PostgreSQLとS3を使うため、ステートレスで問題なし |
| **Persistent** | ⭕ 可 | コンテナ常時稼働、料金は高め |

**PostgreSQLとLeapcell Object Storageを使用するため、どちらのモードでもデータは永続化されます。**

Serverlessモードの方がコスト効率が良いため推奨します。

- デプロイ設定の確認

Leapcellが以下を自動検出しているか確認：

- ✅ **ビルドコマンド**: Dockerfileを使用
- ✅ **ポート**: `8080`
- ✅ **モード**: ServerlessまたはPersistent

- 環境変数を設定

Leapcellダッシュボードの「Environment Variables」セクションで設定：

```text
# JWT
JWT_SECRET=your-generated-jwt-secret-key-at-least-32-bytes
ALLOWED_ORIGINS=https://yourdomain.com
DEBUG=false

# データベース（LeapcellのPostgreSQL接続URL）
DATABASE_URL=postgresql://user:pass@your-db.leapcell.io:5432/taskmanager?sslmode=require

# メール
RESEND_API_KEY=re_xxxxxxxxxxxxx
MAIL_FROM=noreply@yourdomain.com

# ストレージ（Leapcell Object Storage）
STORAGE_TYPE=s3
S3_ENDPOINT=https://objstorage.leapcell.io
S3_REGION=us-east-1
S3_BUCKET=your-bucket-name
S3_ACCESS_KEY=your-s3-access-key-id-here
S3_SECRET_KEY=your-s3-secret-access-key-here
S3_CDN_URL=https://your-cdn-url.leapcellobj.com/your-bucket-name
BASE_URL=https://your-app.leapcell.dev
```

設定方法：

1. 「Add Variable」をクリック
2. `Key`と`Value`を入力
3. 「Save」をクリック

- デプロイ実行

1. 「Deploy」または「Submit」をクリック
2. ビルドログを確認（エラーがないか）
3. デプロイ完了を待つ（通常3-5分）

- デプロイ完了の確認

デプロイが成功すると、URLが表示されます

```text
https://your-app.leapcell.dev
```

動作確認

```bash
# ヘルスチェック
curl https://your-app.leapcell.dev/uploads/

# ユーザー登録テスト
curl -X POST https://your-app.leapcell.dev/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
# 期待: {"message":"Registration successful..."}
```

メール送信確認：

- 登録時に認証コードメールが届く
- Resendダッシュボードで送信履歴を確認
- スパムフォルダも確認

ログ確認（Leapcellダッシュボード → Logs）：

```json
{"level":"INFO","msg":"サーバー起動","port":"8080"}
{"level":"INFO","msg":"メール送信: Resend API を使用します"}
{"level":"INFO","msg":"Resend APIメール送信成功"}
```

#### アーキテクチャ概要

**開発環境（Docker Compose）:**

```text
┌────────────────────────────────────────────┐
│  Docker Compose                             │
│                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │   API    │  │PostgreSQL│  │ MailHog  │ │
│  │ Container│→ │ Container│  │ Container│ │
│  └──────────┘  └──────────┘  └──────────┘ │
│       │              │                      │
│       ↓              ↓                      │
│  ./uploads/    postgres_data               │
│  (volume)         (volume)                 │
└────────────────────────────────────────────┘
```

**本番環境（Leapcell）:**

```text
┌────────────────────────────────────────────┐
│  Leapcell                                   │
│                                             │
│  ┌──────────┐                               │
│  │   API    │                               │
│  │ Container│                               │
│  │(Serverless│                               │
│  │ or Persist)│                              │
│  └──────────┘                               │
│       │                                      │
│       ├→ Leapcell PostgreSQL (managed)      │
│       ├→ Leapcell Object Storage (S3)       │
│       └→ Resend API (email)                 │
└────────────────────────────────────────────┘
```

**利点:**

- ✅ スケーラブル（複数コンテナ対応）
- ✅ データベース独立（どのコンテナからも同じデータ）
- ✅ ファイルストレージ独立（CDN配信）
- ✅ ステートレス設計（Serverlessモード推奨）

#### 環境変数の管理

Leapcellは環境変数を**暗号化して保存**します。`.env`ファイルは不要です。

**環境変数の更新:**

1. Leapcellダッシュボード → プロジェクト → Environment Variables
2. 変更したい変数を編集
3. 「Save」→ **再デプロイが自動実行される**

**セキュリティのベストプラクティス:**

- ✅ 強力な`JWT_SECRET`を使用（32バイト以上）
- ✅ 定期的にローテーション（3-6ヶ月）
- ✅ APIキーは最小権限（Resend: Sending accessのみ）

#### カスタムドメインの設定（オプション）

**1. ドメインの追加:**

1. Leapcellダッシュボード → Domains
2. 「Add Custom Domain」
3. ドメイン名入力（例: `api.yourdomain.com`）

**2. DNS設定:**

ドメインレジストラで以下のレコードを追加：

```text
Type: CNAME
Name: api
Value: your-app.leapcell.dev
TTL: 3600
```

**3. SSL証明書:**

Leapcellが自動的にLet's Encrypt証明書を発行（数分〜数時間）

**4. CORS設定更新:**

環境変数を更新：

```text
ALLOWED_ORIGINS=https://api.yourdomain.com
```

#### トラブルシューティング

**デプロイが失敗する:**

```bash
# ローカルでビルドテスト
docker build --no-cache -t test-app .
docker run -p 8080:8080 test-app

# エラーログ確認
docker logs <container-id>
```

よくある原因：

- Dockerfileのパス間違い
- 依存関係のインストール失敗
- ポート設定のミス

**環境変数が反映されない:**

1. Leapcellダッシュボードで環境変数を確認
2. スペルミスや余分な空白がないか確認
3. 「Redeploy」で再デプロイ
4. ログで確認: `"msg":"メール送信: Resend API を使用します"`

**メールが届かない:**

1. スパムフォルダを確認
2. Resendダッシュボード（Emails → Logs）で送信履歴を確認
3. ドメイン認証を確認（DNS設定）
4. `MAIL_FROM`が認証済みドメインか確認
5. Resendの送信制限に達していないか確認（無料: 100通/日）

**データが消える:**

**症状**: 再デプロイ後にユーザーやタスクが消える

**原因**: データベース接続エラー

**対処**:

1. `DATABASE_URL`が正しく設定されているか確認
2. LeapcellのPostgreSQLデータベースが起動しているか確認
3. ログでデータベース接続エラーを確認

```bash
# Leapcellダッシュボード → Logs で確認
# エラー例: "データベース接続失敗"
```

**画像が表示されない:**

**症状**: アップロードした画像が表示されない

**原因**: Object Storage設定エラー

**対処**:

1. `STORAGE_TYPE=s3`が設定されているか確認
2. S3関連の環境変数が全て設定されているか確認
3. LeapcellのObject Storageが正しく作成されているか確認
4. ログでS3アップロードエラーを確認

```bash
# Leapcellダッシュボード → Logs で確認
# エラー例: "S3アップロードエラー"
```

#### デプロイチェックリスト

**デプロイ前:**

- [ ] GitHubにコードがプッシュされている
- [ ] `.env`ファイルが`.gitignore`で除外されている
- [ ] `Dockerfile`が正しく動作する（ローカルでテスト済み）
- [ ] LeapcellでPostgreSQLデータベースを作成済み
- [ ] LeapcellでObject Storageを作成済み
- [ ] 環境変数を準備（JWT_SECRET、DATABASE_URL、S3設定等）
- [ ] Resendでドメイン認証が完了している
- [ ] CORS設定が正しい（ALLOWED_ORIGINS）

**デプロイ後:**

- [ ] APIエンドポイントが応答する
- [ ] データベース接続が成功している（ログ確認）
- [ ] ユーザー登録が動作する
- [ ] メール送信が動作する
- [ ] 画像アップロードが動作する
- [ ] アップロードした画像が表示される（CDN URL）
- [ ] ログインが動作する
- [ ] ログにエラーがない

#### 定期メンテナンス

**月次:**

- Resendの送信量確認（上限に達していないか）
- ログのエラー確認
- アプリケーションの動作確認

**四半期（3-6ヶ月）:**

  ```bash
# JWT_SECRETのローテーション
openssl rand -base64 32
# Leapcellの環境変数を更新 → 再デプロイ
# 全ユーザーが再ログインする必要あり

# 依存関係のアップデート
go get -u ./...
go mod tidy
```

#### 参考リンク

- [Leapcell公式サイト](https://leapcell.io/)
- [Leapcell: Serverless vs Persistent](https://docs.leapcell.io/ja/serverless-vs-persistent/)
- [Resend公式ドキュメント](https://resend.com/docs)

### オプションB: 手動デプロイ

#### 1. 環境変数の設定

本番環境では以下を**必ず**設定してください：

   ```bash
   # 強力なランダム文字列を生成
   openssl rand -base64 32

   # 環境変数に設定
   export JWT_SECRET="生成された文字列"
   export ALLOWED_ORIGINS="https://yourdomain.com"
   export DEBUG="false"
   ```

#### 2. メール送信サービスの設定（Resend）

本システムは環境に応じて自動的にメール送信方法を切り替えます

#### 環境判定

`RESEND_API_KEY`環境変数の有無で自動判定

- **開発環境**: MailHog（SMTP）を使用 → `RESEND_API_KEY`が未設定
- **本番環境**: Resend APIを使用 → `RESEND_API_KEY`が設定されている

#### 開発環境（MailHog）

**設定不要** - Docker Composeで自動起動されます。

```bash
# Docker Composeで起動
docker compose up -d

# MailHog WebUIにアクセス
open http://localhost:8025
```

#### 本番環境（Resend）

- ステップ1: Resendアカウント作成

[https://resend.com/](https://resend.com/)でアカウントを作成します。

- ステップ2: ドメイン認証

独自ドメインからメール送信するために、ドメイン認証が必要です。

1. Resendダッシュボード → **Domains** → **Add Domain**
2. ドメイン名を入力（例: `yourdomain.com`）
3. DNS設定を確認

    ```text
    追加が必要なDNSレコード
    | Type | Name | Value |
    |------|------|-------|
    | MX  | send | `10 feedback-smtp.ap-nor...` |
    | TXT | send | `v=spf1 include:amazons...` |
    | TXT | resend._domainkey | `p=MIGfMA0GCSqGSIb3DQEB...` |
    ```

4. DNS設定後、**Verify**をクリック
5. 認証完了まで数分〜数時間待機

- ステップ3: APIキー取得

1. Resendダッシュボード → **API Keys** → **Create API Key**
2. 名前を入力（例: `Production Server`）
3. 権限: **Sending access**（送信のみ）
4. **Create**をクリック
5. 表示されたAPIキーをコピー（`re_`で始まる）

**⚠️ 重要**: APIキーは一度しか表示されません。安全に保管してください。

- ステップ4: 環境変数設定

```bash
# 環境変数に設定
export RESEND_API_KEY="re_xxxxxxxxxxxxx"
export MAIL_FROM="noreply@yourdomain.com"
```

Docker Compose（compose.yaml）の場合

```yaml
services:
  api:
    environment:
      - RESEND_API_KEY=${RESEND_API_KEY}
      - MAIL_FROM=noreply@yourdomain.com
```

- ステップ5: 動作確認

サーバー起動時のログで確認

```json
{"level":"INFO","msg":"メール送信: Resend API を使用します"}
```

ユーザー登録時のログ：

```json
{"level":"INFO","msg":"Resend APIでメール送信中","to":"user@example.com"}
{"level":"INFO","msg":"Resend APIメール送信成功","to":"user@example.com","message_id":"..."}
```

#### トラブルシューティング

**エラー: `401 Unauthorized`**

- 原因: APIキーが無効または期限切れ
- 解決策: ResendダッシュボードでAPIキーを確認し、新規発行

**エラー: `403 Forbidden: Domain not verified`**

- 原因: ドメイン認証が未完了
- 解決策: DNS設定を確認し、Resendダッシュボードで**Verify**を再実行

**エラー: `400 Bad Request: Invalid from address`**

- 原因: 送信元メールアドレスが認証済みドメインと一致しない
- 解決策: `MAIL_FROM`を認証済みドメインのアドレスに変更

**メールが届かない:**

1. Resendダッシュボード（Emails → Logs）で送信履歴を確認
2. 迷惑メールフォルダを確認
3. SPF/DKIM/DMARC設定を再確認

#### 料金プラン

| プラン | 月額 | 送信数 | 単価 |
|--------|------|--------|------|
| Free | $0 | 3,000通/月 | 無料 |
| Pro | $20 | 50,000通/月 | $0.0004/通 |
| Scale | カスタム | カスタム | $0.0003/通 |

詳細: [https://resend.com/pricing](https://resend.com/pricing)

#### セキュリティ

**APIキーの管理:**

1. 環境変数で管理（ソースコードにハードコードしない）
2. 最小権限の原則（Sending accessのみ付与）
3. 定期的なローテーション（3-6ヶ月ごとに更新）
4. 漏洩時の対応（すぐに無効化して新規発行）

**送信ドメインの保護:**

1. DMARC設定（`p=quarantine`または`p=reject`）
2. SPF設定（Resendのみ許可）
3. DKIM署名（自動付与）

#### 代替サービス

- SendGrid（実績重視）
- AWS SES（大規模・低コスト）
- Postmark（トランザクショナル特化）

#### 参考リンク

- [Resend公式ドキュメント](https://resend.com/docs)
- [Resend Go SDK](https://github.com/resend/resend-go)
- [ドメイン認証ガイド](https://resend.com/docs/dashboard/domains/introduction)

#### 3. HTTPSの有効化

#### Nginx リバースプロキシ

```nginx
server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### 4. データベースのバックアップ

定期的にバックアップを実施：

```bash
# SQLiteバックアップ
sqlite3 data/tasks.db ".backup data/backup_$(date +%Y%m%d).db"

# cronで自動化（毎日2時）
0 2 * * * cd /path/to/server && sqlite3 data/tasks.db ".backup data/backup_$(date +\%Y\%m\%d).db"
```

---

## ログ

サーバーは標準ライブラリの`log/slog`による構造化ログを出力します。

### 通常モード（INFO）

```bash
go run .
```

### デバッグモード

```bash
DEBUG=true go run .
```

### ログ例

```json
{"time":"2025-10-10T11:30:15Z","level":"INFO","msg":"サーバーを起動しました","port":"8080"}
{"time":"2025-10-10T11:30:20Z","level":"INFO","msg":"リクエスト完了","method":"POST","path":"/auth/login","status":200,"duration_ms":45}
{"time":"2025-10-10T11:30:21Z","level":"INFO","msg":"ログイン成功","user_id":"user-1","email":"test@example.com"}
{"time":"2025-10-10T11:30:30Z","level":"WARN","msg":"ログイン失敗: パスワードが無効","email":"wrong@example.com"}
```

---

## トラブルシューティング

### Docker Compose関連

#### ポート競合

```bash
# ポート8080を使用中のプロセスを確認
lsof -i :8080

# プロセスを停止
kill -9 <PID>

# または compose.yamlでポート変更
# ports:
#   - "8081:8080"
```

#### ビルドエラー

```bash
# キャッシュをクリアして再ビルド
docker compose build --no-cache
docker compose up -d
```

#### 環境変数が読み込まれない

```bash
# .envファイルが存在するか確認
ls -la .env

# ない場合は作成
cp env.example .env

# Docker Composeを再起動
docker compose down
docker compose up -d

# ログで環境変数を確認
docker compose logs api | grep "メール送信"
```

#### JWTトークン生成エラー

```bash
# 強力な秘密鍵を生成
openssl rand -base64 32

# .envファイルに設定
echo "JWT_SECRET=生成された文字列" >> .env

# コンテナ再起動
docker compose restart api
```

#### メールが送信されない（本番環境）

```bash
# .envを確認
cat .env | grep RESEND_API_KEY

# 未設定の場合は追加
echo "RESEND_API_KEY=re_xxxxxxxxxxxxx" >> .env
echo "MAIL_FROM=noreply@yourdomain.com" >> .env

# コンテナ再起動
docker compose restart api

# ログで確認
docker compose logs api | grep "メール送信"
# 出力例: "メール送信: Resend API を使用します"
```

### ローカル実行関連

#### ポートが既に使用されている

```bash
# ポート8080を使用中のプロセスを確認
lsof -i :8080

# プロセスを停止
kill -9 <PID>

# 別のポートで起動
PORT=3000 go run .
```

#### Goモジュールエラー

```bash
# モジュールキャッシュをクリア
go clean -modcache

# 依存関係を再インストール
rm go.sum
go mod tidy
```

### データベース関連

#### データベースリセット

```bash
# Docker Composeの場合（PostgreSQL）
docker compose down -v  # ボリュームも削除
docker compose up -d

# または、PostgreSQLコンテナに接続してリセット
docker compose exec postgres psql -U taskmanager -d taskmanager -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
docker compose restart api
```

### JWT認証エラー

**トークンの有効期限:**

- アクセストークン: 15分
- リフレッシュトークン: 7日

**期限切れの場合:**
`POST /auth/refresh`でトークンを更新してください。

---

## APIテスト

### curlでテスト

```bash
# ユーザー登録
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'

# メール認証（MailHogで認証コードを確認）
curl -X POST http://localhost:8080/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}'

# ログイン
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# タスク一覧取得（要トークン）
curl -X GET http://localhost:8080/tasks \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### テストスクリプトを使用

```bash
chmod +x test_api.sh
./test_api.sh
```

このスクリプトは以下を自動テストします：

- ユーザー登録
- ログイン
- タスク一覧取得
- タスク作成・更新・削除
- タスク完了・未完了
- トークンリフレッシュ

---

## 今後の拡張

### Phase 1: キャッシング ✅ 完了 (PostgreSQL)

- ✅ PostgreSQL移行完了
- ✅ コネクションプーリング最適化済み
- ⭕ Redis統合（検討中）
- ⭕ クエリ結果キャッシュ（検討中）

### Phase 2: ストレージ最適化 ✅ 完了 (Object Storage)

- ✅ S3互換ストレージ対応完了
- ✅ CDN配信対応
- ⭕ 画像最適化（リサイズ・圧縮）
- ⭕ サムネイル生成

### Phase 3: 高度な機能

- WebSocket（リアルタイム更新）
- GraphQL API
- バックグラウンドジョブ処理
- 全文検索（PostgreSQL FTS）

### Phase 4: 運用機能

- ログローテーション
- メトリクス収集（Prometheus）
- アラート設定
- ヘルスチェックエンドポイント拡張

---

## ライセンス

MIT License

---

## サポート

問題が発生した場合は、以下を確認してください：

1. Goのバージョン（1.25.0以上）
2. ポートの競合（8080番ポート）
3. 環境変数の設定（JWT_SECRET、ALLOWED_ORIGINS）
4. データベースファイルのアクセス権限
5. アクセストークンの有効期限
