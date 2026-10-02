package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrBadState 表示 OAuth state 不存在、已过期或已被使用（CSRF 防护失败）。
var ErrBadState = errors.New("invalid oauth state")

// stateTTL 是授权链接的有效期：够用户跳转 GitHub 并点授权，又足够短以限制重放窗口。
const stateTTL = 10 * time.Minute

// StateStore 持久化 OAuth state，实现一次性校验。
// 存表而非内存：多实例部署下 state 也可校验，且进程重启不丢失。
type StateStore struct {
	pool *pgxpool.Pool
}

func NewStateStore(pool *pgxpool.Pool) *StateStore {
	return &StateStore{pool: pool}
}

// Create 生成并保存一个新 state，返回 state 与过期时间。
func (s *StateStore) Create(ctx context.Context, redirect string) (string, time.Time, error) {
	state, err := randomHex(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(stateTTL)

	// 顺带清理过期行，避免表无限增长（无需额外定时任务）。
	if _, err := s.pool.Exec(ctx, `DELETE FROM oauth_states WHERE expires_at < NOW()`); err != nil {
		return "", time.Time{}, fmt.Errorf("purge oauth states: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO oauth_states (state, redirect, expires_at)
		VALUES ($1, NULLIF($2, ''), $3)`, state, redirect, expiresAt); err != nil {
		return "", time.Time{}, fmt.Errorf("insert oauth state: %w", err)
	}
	return state, expiresAt, nil
}

// Consume 校验并「一次性消费」state：命中则原子删除并返回记录的 redirect。
// 已消费的 state 再次使用会返回 ErrBadState，从而阻断授权码重放。
func (s *StateStore) Consume(ctx context.Context, state string) (string, error) {
	if state == "" {
		return "", ErrBadState
	}
	var (
		redirect  *string
		expiresAt time.Time
	)
	// DELETE ... RETURNING 保证「校验+删除」是一个原子操作，并发重放只有一个成功。
	err := s.pool.QueryRow(ctx, `
		DELETE FROM oauth_states WHERE state = $1
		RETURNING redirect, expires_at`, state).Scan(&redirect, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrBadState
	}
	if err != nil {
		return "", fmt.Errorf("consume oauth state: %w", err)
	}
	if time.Now().After(expiresAt) {
		return "", ErrBadState
	}
	if redirect == nil {
		return "", nil
	}
	return *redirect, nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
