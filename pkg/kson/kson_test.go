package kson_test

import (
	"testing"

	"github.com/fluxorio/fluxor/pkg/kson"
)

const sample = `
# kson sample config

HOST     = localhost
PORT     = 8080
DEBUG    = false
RATIO    = 3.14
NAME     = "Saving Bank Platform"

# dotted paths
db.host     = localhost
db.port     = 5432
db.pool.max = 8
db.pool.min = 2

# inline object
smtp = { host: smtpdm.aliyun.com, port: 465, ssl: true }

# array
features = [auth, wallet, orders]
ports    = [8080, 8081, 8082]

# quoted with spaces and special chars
dsn = "postgres://user:pass@localhost/db?sslmode=disable"
`

func TestParse_Scalars(t *testing.T) {
	m, err := kson.Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if got := kson.GetString(m, "HOST", ""); got != "localhost" {
		t.Errorf("HOST = %q, want %q", got, "localhost")
	}
	if got := kson.GetInt(m, "PORT", 0); got != 8080 {
		t.Errorf("PORT = %d, want 8080", got)
	}
	if got := kson.GetBool(m, "DEBUG", true); got != false {
		t.Errorf("DEBUG = %v, want false", got)
	}
	if got := kson.GetString(m, "NAME", ""); got != "Saving Bank Platform" {
		t.Errorf("NAME = %q", got)
	}
}

func TestParse_DottedPaths(t *testing.T) {
	m, err := kson.Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if got := kson.GetString(m, "db.host", ""); got != "localhost" {
		t.Errorf("db.host = %q", got)
	}
	if got := kson.GetInt(m, "db.port", 0); got != 5432 {
		t.Errorf("db.port = %d", got)
	}
	if got := kson.GetInt(m, "db.pool.max", 0); got != 8 {
		t.Errorf("db.pool.max = %d", got)
	}
}

func TestParse_InlineObject(t *testing.T) {
	m, err := kson.Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	smtp := kson.GetMap(m, "smtp")
	if smtp == nil {
		t.Fatal("smtp map is nil")
	}
	if got := kson.GetString(smtp, "host", ""); got != "smtpdm.aliyun.com" {
		t.Errorf("smtp.host = %q", got)
	}
	if got := kson.GetInt(smtp, "port", 0); got != 465 {
		t.Errorf("smtp.port = %d", got)
	}
	if got := kson.GetBool(smtp, "ssl", false); !got {
		t.Errorf("smtp.ssl = false, want true")
	}
}

func TestParse_Array(t *testing.T) {
	m, err := kson.Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	v := kson.Get(m, "features")
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("features is %T, want []any", v)
	}
	if len(arr) != 3 {
		t.Errorf("features len = %d, want 3", len(arr))
	}
	if arr[0] != "auth" {
		t.Errorf("features[0] = %v, want auth", arr[0])
	}
}

func TestParse_QuotedDSN(t *testing.T) {
	m, err := kson.Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	want := "postgres://user:pass@localhost/db?sslmode=disable"
	if got := kson.GetString(m, "dsn", ""); got != want {
		t.Errorf("dsn = %q, want %q", got, want)
	}
}

func TestParse_KeyConflict(t *testing.T) {
	src := "db = localhost\ndb.port = 5432\n"
	_, err := kson.Parse([]byte(src))
	if err == nil {
		t.Fatal("expected error for key conflict")
	}
}

func TestParse_MalformedNoEquals(t *testing.T) {
	_, err := kson.Parse([]byte("HOST localhost\n"))
	if err == nil {
		t.Fatal("expected error for missing '='")
	}
}

func BenchmarkParse(b *testing.B) {
	src := []byte(sample)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := kson.Parse(src); err != nil {
			b.Fatal(err)
		}
	}
}
