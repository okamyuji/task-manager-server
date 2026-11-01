package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
)

// getJWTSecret は JWT_SECRET が空だと os.Exit(1) するため、テスト開始前に実行時生成の値を入れる。
func TestMain(m *testing.M) {
	if os.Getenv("JWT_SECRET") == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic(err)
		}
		_ = os.Setenv("JWT_SECRET", hex.EncodeToString(b))
	}
	os.Exit(m.Run())
}
