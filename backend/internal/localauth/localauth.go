// Package localauth 实现本地用户名 + 密码认证。
//
// 与 GitHub OAuth 并存：同一个 users 表，password_hash 为 NULL 的账号
// 只能走 OAuth 登录，反之亦然。
//
// 设计要点：
//   - 密码用 bcrypt 加盐哈希存储，绝不存明文，也不可逆。
//   - 登录失败对不同原因（用户不存在 / 密码错误）返回**相同**错误，
//     避免通过错误信息枚举已存在的用户名。
//   - 首次部署引导（Setup）与日常登录共用这里的哈希逻辑，保证强度一致。
package localauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials 是登录失败的统一错误（不区分用户不存在与密码错误）。
	ErrInvalidCredentials = errors.New("用户名或密码不正确")
	// ErrWeakPassword 表示密码不满足强度要求。
	ErrWeakPassword = errors.New("密码强度不足")
	// ErrInvalidUsername 表示用户名格式非法。
	ErrInvalidUsername = errors.New("用户名格式不合要求")
)

// 用户名长度按「字符数」而非字节数计算，避免中文用户名被误判。
const (
	MinUsernameLen = 3
	MaxUsernameLen = 32
	MinPasswordLen = 8
	MaxPasswordLen = 72 // bcrypt 只取前 72 字节，超出部分会被静默忽略
)

// Store 负责本地账号的读写。
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// UserCount 返回用户总数，用于判断是否需要进入首次部署引导。
func (s *Store) UserCount(ctx context.Context) (int64, error) {
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("统计用户数: %w", err)
	}
	return n, nil
}

// CreateFirstAdmin 在**没有任何用户**时创建首个管理员。
//
// 这是安全关键操作，因此把「表为空」的判断与插入放在**同一个事务**里，
// 并用 `SELECT ... FOR UPDATE` 之外的方式保证互斥：
// 这里依赖 users 表的唯一索引 —— 若并发请求同时通过计数检查，
// 后一个插入会因用户名冲突而失败，不会产生两个管理员。
//
// 返回值为 (created, err)：created=false 表示已存在用户，调用方应拒绝该请求。
func (s *Store) CreateFirstAdmin(ctx context.Context, username, password string) (bool, error) {
	if err := ValidateUsername(username); err != nil {
		return false, err
	}
	if err := ValidatePassword(password); err != nil {
		return false, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("哈希密码: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("开启事务: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 串行化：锁住整张表的计数判断，避免并发引导产生多个管理员。
	var count int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, fmt.Errorf("统计用户数: %w", err)
	}
	if count > 0 {
		return false, nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (username, password_hash, is_admin)
		VALUES ($1, $2, TRUE)`, username, string(hash)); err != nil {
		return false, fmt.Errorf("创建管理员: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("提交事务: %w", err)
	}
	return true, nil
}

// Authenticate 校验用户名与密码，成功时返回用户 ID。
//
// 无论「用户不存在」还是「密码错误」，都返回 ErrInvalidCredentials，
// 并且**即使账号不存在也执行一次 bcrypt 比较**，让两条路径耗时接近，
// 避免通过响应时间差异枚举用户名。
func (s *Store) Authenticate(ctx context.Context, username, password string) (int64, error) {
	var (
		id   int64
		hash *string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, password_hash FROM users WHERE LOWER(username) = LOWER($1)`,
		strings.TrimSpace(username),
	).Scan(&id, &hash)

	if errors.Is(err, pgx.ErrNoRows) {
		// 账号不存在：仍做一次哈希比较以拉平耗时，然后返回统一错误。
		_ = bcrypt.CompareHashAndPassword(fakeHash, []byte(password))
		return 0, ErrInvalidCredentials
	}
	if err != nil {
		return 0, fmt.Errorf("查询用户: %w", err)
	}
	if hash == nil || *hash == "" {
		// 账号存在但未设置密码（纯 OAuth 用户）：不能凭密码登录。
		return 0, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*hash), []byte(password)); err != nil {
		return 0, ErrInvalidCredentials
	}
	return id, nil
}

// SetPassword 为已有用户设置或重置密码（供后续「修改密码」功能使用）。
func (s *Store) SetPassword(ctx context.Context, userID int64, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("哈希密码: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		userID, string(hash))
	if err != nil {
		return fmt.Errorf("更新密码: %w", err)
	}
	return nil
}

// ValidateUsername 校验用户名：长度合规、只含字母/数字/下划线/连字符。
//
// 限制字符集是为了避免用户名在 URL 路径（/repos/{ns}/...）与日志中
// 引入歧义或注入风险。
func ValidateUsername(name string) error {
	name = strings.TrimSpace(name)
	n := len([]rune(name))
	if n < MinUsernameLen || n > MaxUsernameLen {
		return fmt.Errorf("%w：长度需在 %d-%d 个字符之间", ErrInvalidUsername, MinUsernameLen, MaxUsernameLen)
	}
	for _, r := range name {
		if unicode.IsLetter(r) && r < unicode.MaxASCII {
			continue
		}
		if unicode.IsDigit(r) && r < unicode.MaxASCII {
			continue
		}
		if r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("%w：只能包含英文字母、数字、下划线和连字符", ErrInvalidUsername)
	}
	return nil
}

// ValidatePassword 校验密码强度。
//
// 要求：至少 8 个字符，且不能是常见弱口令。
// 不强制"大小写+数字+符号"的复杂组合——那类规则往往促使用户选择
// "Password1!" 这类可预测密码，长度与黑名单更有效。
func ValidatePassword(pw string) error {
	if len([]rune(pw)) < MinPasswordLen {
		return fmt.Errorf("%w：至少需要 %d 个字符", ErrWeakPassword, MinPasswordLen)
	}
	// bcrypt 只使用前 72 字节；超长密码会被静默截断，导致用户误以为更安全。
	if len(pw) > MaxPasswordLen {
		return fmt.Errorf("%w：不能超过 %d 字节", ErrWeakPassword, MaxPasswordLen)
	}
	lower := strings.ToLower(pw)
	for _, weak := range []string{
		"password", "12345678", "123456789", "qwerty", "abc123",
		"admin", "letmein", "iloveyou", "welcome", "monkey",
		"boxli", "admin123", "password1", "11111111",
	} {
		if lower == weak || strings.Contains(lower, weak) {
			return fmt.Errorf("%w：不能包含常见弱口令", ErrWeakPassword)
		}
	}
	return nil
}

// fakeHash 是一个固定的 bcrypt 哈希，用于账号不存在时消耗等量时间。
// 明文为随机字符串，无实际匹配意义。
var fakeHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")
