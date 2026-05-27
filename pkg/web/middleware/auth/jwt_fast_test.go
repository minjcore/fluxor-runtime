package auth_test

import (
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "postgres-secret-key-change-in-production"

// makeToken generates a valid HS256 JWT using golang-jwt (the "old" path).
func makeToken(secret string, username string, ttl time.Duration) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(ttl).Unix(),
		"iat":      time.Now().Unix(),
	})
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	return s
}

func TestFastJWT_ValidToken(t *testing.T) {
	tok := makeToken(testSecret, "khangdc", time.Hour)
	v := auth.NewFastJWTVerifier(testSecret)
	claims, err := v.Verify(tok)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if claims.Username != "khangdc" {
		t.Fatalf("username = %q, want %q", claims.Username, "khangdc")
	}
}

func TestFastJWT_ExpiredToken(t *testing.T) {
	tok := makeToken(testSecret, "khangdc", -time.Second)
	v := auth.NewFastJWTVerifier(testSecret)
	_, err := v.Verify(tok)
	if err != auth.ErrJWTExpired {
		t.Fatalf("want ErrJWTExpired, got %v", err)
	}
}

func TestFastJWT_BadSignature(t *testing.T) {
	tok := makeToken(testSecret, "khangdc", time.Hour)
	// Tamper with last char of signature
	tok = tok[:len(tok)-1] + "X"
	v := auth.NewFastJWTVerifier(testSecret)
	_, err := v.Verify(tok)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestFastJWT_WrongSecret(t *testing.T) {
	tok := makeToken("different-secret", "khangdc", time.Hour)
	v := auth.NewFastJWTVerifier(testSecret)
	_, err := v.Verify(tok)
	if err != auth.ErrJWTBadSig {
		t.Fatalf("want ErrJWTBadSig, got %v", err)
	}
}

func TestFastJWT_MalformedToken(t *testing.T) {
	v := auth.NewFastJWTVerifier(testSecret)
	for _, bad := range []string{"", "notajwt", "a.b", "a.b.", "a..c"} {
		if _, err := v.Verify(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

// BenchmarkFastJWT_Verify benchmarks the new parser.
func BenchmarkFastJWT_Verify(b *testing.B) {
	tok := makeToken(testSecret, "khangdc", time.Hour)
	v := auth.NewFastJWTVerifier(testSecret)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := v.Verify(tok); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOldJWT_ParseWithClaims benchmarks the golang-jwt baseline.
func BenchmarkOldJWT_ParseWithClaims(b *testing.B) {
	tok := makeToken(testSecret, "khangdc", time.Hour)
	key := []byte(testSecret)
	keyFunc := func(t *jwt.Token) (interface{}, error) { return key, nil }
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := jwt.ParseWithClaims(tok, jwt.MapClaims{}, keyFunc,
			jwt.WithValidMethods([]string{"HS256"})); err != nil {
			b.Fatal(err)
		}
	}
}
