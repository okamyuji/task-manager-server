package repository

import (
	"testing"
	"time"
)

// TestVerificationRepository_Create_正常系
func TestVerificationRepository_Create_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	verifyRepo := NewVerificationRepository(db)

	user := &User{
		Email:        "verify@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	code, err := verifyRepo.Create(user.ID, 15*time.Minute)
	assertNoError(t, err)

	if code.ID == 0 {
		t.Error("Create() ID が設定されていません")
	}
	if code.Code == "" {
		t.Error("Create() Code が設定されていません")
	}
}

// TestVerificationRepository_Create_異常系_存在しないユーザー
func TestVerificationRepository_Create_異常系_存在しないユーザー(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	verifyRepo := NewVerificationRepository(db)

	_, err := verifyRepo.Create("nonexistent-user", 15*time.Minute)
	assertError(t, err)
}

// TestVerificationRepository_GetByUserIDAndCode_正常系
func TestVerificationRepository_GetByUserIDAndCode_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	verifyRepo := NewVerificationRepository(db)

	user := &User{
		Email:        "getbycode@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	code, err := verifyRepo.Create(user.ID, 15*time.Minute)
	assertNoError(t, err)

	retrieved, err := verifyRepo.GetByUserIDAndCode(user.ID, code.Code)
	assertNoError(t, err)

	assertEqual(t, retrieved.Code, code.Code)
	assertEqual(t, retrieved.UserID, user.ID)
}

// TestVerificationRepository_GetByUserIDAndCode_異常系_存在しないコード
func TestVerificationRepository_GetByUserIDAndCode_異常系_存在しないコード(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	verifyRepo := NewVerificationRepository(db)

	user := &User{
		Email:        "test@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	code, err := verifyRepo.GetByUserIDAndCode(user.ID, "nonexistent-code")
	assertNoError(t, err)
	if code != nil {
		t.Errorf("expected nil code, got %v", code)
	}
}

// TestVerificationRepository_MarkAsUsed_正常系
func TestVerificationRepository_MarkAsUsed_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	verifyRepo := NewVerificationRepository(db)

	user := &User{
		Email:        "markused@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	code, err := verifyRepo.Create(user.ID, 15*time.Minute)
	assertNoError(t, err)

	err = verifyRepo.MarkAsUsed(code.ID)
	assertNoError(t, err)

	usedCode, err := verifyRepo.GetByUserIDAndCode(user.ID, code.Code)
	assertNoError(t, err) // エラーは返さない
	if usedCode != nil {
		t.Errorf("expected nil (used code should not be returned), got %v", usedCode)
	}
}

// TestVerificationRepository_DeleteExpired_正常系
func TestVerificationRepository_DeleteExpired_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	verifyRepo := NewVerificationRepository(db)

	user := &User{
		Email:        "deleteexp@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	// 有効なコード
	validCode, err := verifyRepo.Create(user.ID, 15*time.Minute)
	assertNoError(t, err)

	// 期限切れコード（直接DBに挿入）
	query := `INSERT INTO verification_codes (user_id, code, expires_at, created_at, used) VALUES ($1, $2, $3, $4, $5)`
	expiredTime := time.Now().Add(-1 * time.Hour)
	_, err = db.Exec(query, user.ID, "EXPIRED123", expiredTime, time.Now(), false)
	assertNoError(t, err)

	// 期限切れコードを削除
	err = verifyRepo.DeleteExpired()
	assertNoError(t, err)

	// 有効なコードは取得できる
	_, err = verifyRepo.GetByUserIDAndCode(user.ID, validCode.Code)
	assertNoError(t, err)

	// 期限切れコードは取得できない
	expiredCode, err := verifyRepo.GetByUserIDAndCode(user.ID, "EXPIRED123")
	assertNoError(t, err)
	if expiredCode != nil {
		t.Errorf("expected nil (expired code should be deleted), got %v", expiredCode)
	}
}

// TestVerificationRepository_DeleteByUserID_正常系
func TestVerificationRepository_DeleteByUserID_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	verifyRepo := NewVerificationRepository(db)

	user := &User{
		Email:        "deletebyuser@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	// 複数のコードを作成
	var codes []*VerificationCode
	for i := 0; i < 3; i++ {
		code, err := verifyRepo.Create(user.ID, 15*time.Minute)
		assertNoError(t, err)
		codes = append(codes, code)
	}

	// ユーザーのコードを全削除
	err := verifyRepo.DeleteByUserID(user.ID)
	assertNoError(t, err)

	// 全てのコードが取得できないことを確認
	for _, code := range codes {
		deletedCode, err := verifyRepo.GetByUserIDAndCode(user.ID, code.Code)
		assertNoError(t, err)
		if deletedCode != nil {
			t.Errorf("expected nil (code should be deleted), got %v", deletedCode)
		}
	}
}
