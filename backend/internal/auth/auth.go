package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoSecret   = errors.New("jwt secret not configured")
	ErrBadToken   = errors.New("invalid token")
	ErrExpired    = errors.New("session expired")
	ErrRevoked    = errors.New("session revoked")
	ErrNoSession  = errors.New("session not found")
	ErrNoDBRecord = errors.New("session record not found")
)

// User 是从会话解析出的当前用户。
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type Claims struct {
	UserID int64  `json:"uid"`
	SID    string `json:"sid"`
	jwt.RegisteredClaims
}

// Service 负责 JWT 签发、校验与 sessions 表读写。
type Service struct {
	pool   *pgxpool.Pool
	secret []byte
	ttl    time.Duration
}

func New(pool *pgxpool.Pool, secret string, ttlHours int) *Service {
	return &Service{pool: pool, secret: []byte(secret), ttl: time.Duration(ttlHours) * time.Hour}
}

// Enabled 表示是否配置了签名密钥。
func (s *Service) Enabled() bool { return len(s.secret) > 0 }

// Issue 生成 JWT 并把 token 哈希写入 sessions 表，返回 token 与过期时间。
func (s *Service) Issue(ctx context.Context, userID int64) (string, time.Time, error) {
	if !s.Enabled() {
		return "", time.Time{}, ErrNoSecret
	}
	sid, err := randomHex(16)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(s.ttl)

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`,
		sid, userID, hash(sid), expiresAt); err != nil {
		return "", time.Time{}, fmt.Errorf("insert session: %w", err)
	}

	claims := Claims{
		UserID: userID,
		SID:    sid,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprint(userID),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return token, expiresAt, nil
}

// Verify 校验 JWT 并确认对应 session 仍有效，返回当前用户与 session id。
func (s *Service) Verify(ctx context.Context, token string) (*User, string, error) {
	if !s.Enabled() {
		return nil, "", ErrNoSecret
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrBadToken
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, "", ErrExpired
		}
		return nil, "", ErrBadToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, "", ErrBadToken
	}

	var (
		gotHash   string
		expiresAt time.Time
	)
	err = s.pool.QueryRow(ctx, `
		SELECT token_hash, expires_at FROM sessions WHERE id = $1`, claims.SID,
	).Scan(&gotHash, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNoSession
	}
	if err != nil {
		return nil, "", fmt.Errorf("load session: %w", err)
	}
	if gotHash != hash(claims.SID) {
		return nil, "", ErrBadToken
	}
	if time.Now().After(expiresAt) {
		return nil, "", ErrExpired
	}

	var u User
	err = s.pool.QueryRow(ctx, `
		SELECT id, username, COALESCE(email, ''), COALESCE(avatar_url, '')
		FROM users WHERE id = $1`, claims.UserID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNoSession
	}
	if err != nil {
		return nil, "", fmt.Errorf("load user: %w", err)
	}
	return &u, claims.SID, nil
}

// Revoke 删除指定 session（登出）。
func (s *Service) Revoke(ctx context.Context, sid string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sid); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
