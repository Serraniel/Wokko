package api_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/api"
	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/config"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"
)

const defaultTestDBURL = "postgres://wokko:changeme@localhost:5432/wokko_test?sslmode=disable"

func testDBURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return defaultTestDBURL
}

// setupTestDB connects to a real PostgreSQL database, runs migrations, and
// returns the pool together with a cleanup func that truncates all data tables.
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := testDBURL()
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("no database available (%v) – skipping integration test", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no database available (%v) – skipping integration test", err)
	}

	// Run migrations (path relative to backend root; tests run from the package dir)
	m, err := migrate.New("file://../../migrations", dbURL)
	if err != nil {
		pool.Close()
		t.Fatalf("create migrator: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		m.Close()
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	m.Close()

	cleanup := func() {
		// Truncate in dependency order so FK constraints are satisfied.
		tables := []string{
			"step_ingredients",
			"recipe_steps",
			"recipe_tags",
			"share_links",
			"mcp_tokens",
			"passkey_credentials",
			"refresh_tokens",
			"ingredient_unit_stats",
			"ingredient_aliases",
			"invite_links",
			"recipes",
			"ingredients",
			"units",
			"unit_aliases",
			"tags",
			"users",
			"system_settings",
		}
		for _, tbl := range tables {
			_, _ = pool.Exec(context.Background(), fmt.Sprintf("TRUNCATE %s CASCADE", tbl))
		}
		pool.Close()
	}

	return pool, cleanup
}

// testCfg returns a minimal *config.Config suitable for tests.
func testCfg(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		JWTSecret:          "test-jwt-secret-for-integration",
		UploadPath:         t.TempDir(),
		PublicURL:          "http://localhost",
		Port:               "8080",
		OpenRegistration:   false,
		AllowPublicRecipes: true,
	}
}

// createTestUser inserts a user directly into the database and returns the
// user's UUID string and a freshly-minted JWT access token.
func createTestUser(t *testing.T, pool *pgxpool.Pool, cfg *config.Config, role string) (userID string, token string) {
	t.Helper()

	ctx := context.Background()
	q := dbq.New(pool)

	username := fmt.Sprintf("testuser_%s_%d", role, time.Now().UnixNano())
	hash, err := auth.HashPassword("testpassword123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user, err := q.CreateUser(ctx, dbq.CreateUserParams{
		Username:     username,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
		Role:         role,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	uid, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		t.Fatalf("parse user UUID: %v", err)
	}

	tok, err := auth.GenerateAccessToken(cfg.JWTSecret, uid, user.Username, user.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	return uid.String(), tok
}

// newIntegrationServer creates a real api.Server backed by pool/cfg, registers
// all routes, and wraps it in an httptest.Server.  The caller is responsible
// for calling ts.Close().
func newIntegrationServer(t *testing.T, pool *pgxpool.Pool, cfg *config.Config) *httptest.Server {
	t.Helper()

	e := echo.New()
	api.RegisterRoutes(e, pool, cfg)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return ts
}
