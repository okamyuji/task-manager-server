package handlers

import (
	"context"
	"errors"
	"time"

	"task_manager_server/database/repository"
	"task_manager_server/utils"
)

// mockUserRepository
type mockUserRepository struct {
	users map[string]*repository.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*repository.User),
	}
}

func (m *mockUserRepository) Create(user *repository.User) error {
	if user.ID == "" {
		user.ID = "mock-user-id"
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepository) GetByID(id string) (*repository.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) GetByEmail(email string) (*repository.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *mockUserRepository) Update(user *repository.User) error {
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepository) Delete(id string) error {
	for email, u := range m.users {
		if u.ID == id {
			delete(m.users, email)
			return nil
		}
	}
	return nil
}

func (m *mockUserRepository) VerifyUser(id string) error {
	for _, u := range m.users {
		if u.ID == id {
			u.IsVerified = true
			return nil
		}
	}
	return nil
}

func (m *mockUserRepository) List() ([]*repository.User, error) {
	var result []*repository.User
	for _, u := range m.users {
		result = append(result, u)
	}
	return result, nil
}

// mockVerificationRepository
type mockVerificationRepository struct {
	codes  map[int64]*repository.VerificationCode
	nextID int64
}

func newMockVerificationRepository() *mockVerificationRepository {
	return &mockVerificationRepository{
		codes:  make(map[int64]*repository.VerificationCode),
		nextID: 1,
	}
}

func (m *mockVerificationRepository) Create(userID string, expiresIn time.Duration) (*repository.VerificationCode, error) {
	code := &repository.VerificationCode{
		ID:        m.nextID,
		UserID:    userID,
		Code:      "123456",
		ExpiresAt: time.Now().Add(expiresIn),
		CreatedAt: time.Now(),
		Used:      false,
	}
	m.codes[m.nextID] = code
	m.nextID++
	return code, nil
}

func (m *mockVerificationRepository) GetByUserIDAndCode(userID, code string) (*repository.VerificationCode, error) {
	for _, c := range m.codes {
		if c.UserID == userID && c.Code == code && !c.Used {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockVerificationRepository) MarkAsUsed(id int64) error {
	if c, ok := m.codes[id]; ok {
		c.Used = true
		return nil
	}
	return nil
}

func (m *mockVerificationRepository) DeleteExpired() error {
	return nil
}

func (m *mockVerificationRepository) DeleteByUserID(userID string) error {
	for id, c := range m.codes {
		if c.UserID == userID {
			delete(m.codes, id)
		}
	}
	return nil
}

// mockEmailService
type mockEmailService struct {
	sentEmails []string
}

func (m *mockEmailService) SendVerificationCode(ctx context.Context, to, name, code string) error {
	m.sentEmails = append(m.sentEmails, to)
	return nil
}

func (m *mockEmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	m.sentEmails = append(m.sentEmails, to)
	return nil
}

// mockTokenService
type mockTokenService struct{}

func (m *mockTokenService) GenerateAccessToken(userID, email string) (string, error) {
	return "mock-access-token", nil
}

func (m *mockTokenService) GenerateRefreshToken(userID, email string) (string, error) {
	return "mock-refresh-token", nil
}

func (m *mockTokenService) VerifyToken(tokenString string) (*utils.JWTClaims, error) {
	// どのトークンでも正常に検証する（テスト用）
	return &utils.JWTClaims{
		UserID: "user123",
		Email:  "refresh@example.com",
	}, nil
}

// mockPasswordHasher
type mockPasswordHasher struct{}

func (m *mockPasswordHasher) HashPassword(password string) (string, error) {
	return "hashed-" + password, nil
}

func (m *mockPasswordHasher) ComparePassword(hashedPassword, password string) error {
	if hashedPassword == "hashed-"+password {
		return nil
	}
	return errors.New("password mismatch")
}
