package repository

import (
	"testing"
	"time"
)

// TestUserRepository_Create_正常系
func TestUserRepository_Create_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	user := &User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		Name:         "Test User",
		IsVerified:   false,
	}

	err := repo.Create(user)
	assertNoError(t, err)

	// IDが自動生成されていることを確認
	if user.ID == "" {
		t.Error("Create() user.ID が空です")
	}

	// CreatedAt, UpdatedAtが設定されていることを確認
	if user.CreatedAt.IsZero() {
		t.Error("Create() user.CreatedAt が設定されていません")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("Create() user.UpdatedAt が設定されていません")
	}
}

// TestUserRepository_Create_異常系_重複メール
func TestUserRepository_Create_異常系_重複メール(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	user1 := &User{
		Email:        "duplicate@example.com",
		PasswordHash: "hashed_password",
		Name:         "User 1",
	}
	assertNoError(t, repo.Create(user1))

	// 同じメールアドレスで再度作成
	user2 := &User{
		Email:        "duplicate@example.com",
		PasswordHash: "hashed_password",
		Name:         "User 2",
	}
	err := repo.Create(user2)
	assertError(t, err)
}

// TestUserRepository_GetByID_正常系
func TestUserRepository_GetByID_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	// ユーザー作成
	user := &User{
		Email:        "getbyid@example.com",
		PasswordHash: "hashed_password",
		Name:         "Get By ID User",
	}
	assertNoError(t, repo.Create(user))

	// ID で取得
	retrieved, err := repo.GetByID(user.ID)
	assertNoError(t, err)

	assertEqual(t, retrieved.ID, user.ID)
	assertEqual(t, retrieved.Email, user.Email)
	assertEqual(t, retrieved.Name, user.Name)
}

// TestUserRepository_GetByID_異常系_存在しないID
func TestUserRepository_GetByID_異常系_存在しないID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	user, err := repo.GetByID("nonexistent-id")
	assertNoError(t, err)
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
}

// TestUserRepository_GetByEmail_正常系
func TestUserRepository_GetByEmail_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	// ユーザー作成
	user := &User{
		Email:        "getbyemail@example.com",
		PasswordHash: "hashed_password",
		Name:         "Get By Email User",
	}
	assertNoError(t, repo.Create(user))

	// Emailで取得
	retrieved, err := repo.GetByEmail(user.Email)
	assertNoError(t, err)

	assertEqual(t, retrieved.Email, user.Email)
	assertEqual(t, retrieved.Name, user.Name)
}

// TestUserRepository_GetByEmail_異常系_存在しないEmail
func TestUserRepository_GetByEmail_異常系_存在しないEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	user, err := repo.GetByEmail("nonexistent@example.com")
	assertNoError(t, err)
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
}

// TestUserRepository_Update_正常系
func TestUserRepository_Update_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	// ユーザー作成
	user := &User{
		Email:        "update@example.com",
		PasswordHash: "old_password",
		Name:         "Old Name",
	}
	assertNoError(t, repo.Create(user))

	// 更新
	user.Name = "New Name"
	user.PasswordHash = "new_password"
	err := repo.Update(user)
	assertNoError(t, err)

	// 確認
	retrieved, err := repo.GetByID(user.ID)
	assertNoError(t, err)
	assertEqual(t, retrieved.Name, "New Name")
	assertEqual(t, retrieved.PasswordHash, "new_password")
}

// TestUserRepository_Update_異常系_存在しないユーザー
func TestUserRepository_Update_異常系_存在しないユーザー(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	user := &User{
		ID:           "nonexistent-id",
		Email:        "test@example.com",
		PasswordHash: "password",
		Name:         "Test",
	}
	err := repo.Update(user)
	assertError(t, err)
}

// TestUserRepository_Delete_正常系
func TestUserRepository_Delete_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	// ユーザー作成
	user := &User{
		Email:        "delete@example.com",
		PasswordHash: "password",
		Name:         "Delete User",
	}
	assertNoError(t, repo.Create(user))

	// 削除
	err := repo.Delete(user.ID)
	assertNoError(t, err)

	// 確認（存在しないことを確認）
	deleted, err := repo.GetByID(user.ID)
	assertNoError(t, err)
	if deleted != nil {
		t.Errorf("expected nil user after delete, got %v", deleted)
	}
}

// TestUserRepository_VerifyUser_正常系
func TestUserRepository_VerifyUser_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	// 未認証ユーザー作成
	user := &User{
		Email:        "verify@example.com",
		PasswordHash: "password",
		Name:         "Verify User",
		IsVerified:   false,
	}
	assertNoError(t, repo.Create(user))

	// 認証
	err := repo.VerifyUser(user.ID)
	assertNoError(t, err)

	// 確認
	retrieved, err := repo.GetByID(user.ID)
	assertNoError(t, err)
	if !retrieved.IsVerified {
		t.Error("VerifyUser() ユーザーが認証されていません")
	}
}

// TestUserRepository_List_正常系
func TestUserRepository_List_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	// 複数ユーザー作成
	for i := 0; i < 3; i++ {
		user := &User{
			Email:        "list" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "password",
			Name:         "List User",
		}
		assertNoError(t, repo.Create(user))
		time.Sleep(time.Millisecond) // CreatedAtに差をつける
	}

	// 一覧取得
	users, err := repo.List()
	assertNoError(t, err)

	if len(users) < 3 {
		t.Errorf("List() ユーザー数が不正: got %d, want >= 3", len(users))
	}
}

// TestUserRepository_List_空の場合
func TestUserRepository_List_空の場合(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	repo := NewUserRepository(db)

	users, err := repo.List()
	assertNoError(t, err)

	if len(users) != 0 {
		t.Errorf("List() 空のはずなのにユーザーが存在: got %d", len(users))
	}
}
