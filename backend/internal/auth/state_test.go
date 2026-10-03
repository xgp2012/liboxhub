package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LiStudioorg/boxli/internal/config"
)

// testPool 用 backend/hub.toml 里的 db.url 连接开发库；连不上则跳过
// （保证 CI / 无库环境下测试不会失败）。
// 需要指向别的库时改 hub.toml 的 [db].url。
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := config.Load("../../hub.toml")
	if err != nil {
		t.Skipf("no config: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	return pool
}

// TestStateConsumeIsSingleUse 验证 state 只能被消费一次（防授权码重放）。
func TestStateConsumeIsSingleUse(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	st := NewStateStore(pool)

	state, expiresAt, err := st.Create(ctx, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if state == "" {
		t.Fatal("Create returned empty state")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("state already expired: %v", expiresAt)
	}

	// 首次消费成功
	if _, err := st.Consume(ctx, state); err != nil {
		t.Fatalf("first Consume should succeed, got %v", err)
	}
	// 二次消费必须失败：这是 CSRF / 重放防护的核心断言
	if _, err := st.Consume(ctx, state); !errors.Is(err, ErrBadState) {
		t.Fatalf("replayed state must be rejected, got %v", err)
	}
}

// TestStateRejectsUnknownAndEmpty 验证伪造/空 state 被拒绝。
func TestStateRejectsUnknownAndEmpty(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	st := NewStateStore(pool)

	for _, tc := range []struct{ name, state string }{
		{"empty", ""},
		{"forged", "deadbeefdeadbeefdeadbeefdeadbeef"},
	} {
		if _, err := st.Consume(ctx, tc.state); !errors.Is(err, ErrBadState) {
			t.Errorf("%s state must be rejected, got %v", tc.name, err)
		}
	}
}

// TestStateExpiredRejected 验证过期 state 被拒绝（即使行仍在表中）。
func TestStateExpiredRejected(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	st := NewStateStore(pool)

	state, err := randomHex(32)
	if err != nil {
		t.Fatalf("randomHex: %v", err)
	}
	// 直接插入一条已过期的 state
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_states (state, expires_at) VALUES ($1, NOW() - INTERVAL '1 minute')`,
		state); err != nil {
		t.Fatalf("insert expired state: %v", err)
	}
	if _, err := st.Consume(ctx, state); !errors.Is(err, ErrBadState) {
		t.Fatalf("expired state must be rejected, got %v", err)
	}
	// 过期行应已被 Consume 删除
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_states WHERE state = $1`, state).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("expired state row should be deleted, found %d", n)
	}
}
