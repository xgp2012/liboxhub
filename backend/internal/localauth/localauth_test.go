package localauth

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		valid bool
	}{
		{"普通用户名", "alice", true},
		{"含数字", "user123", true},
		{"含下划线连字符", "my_user-1", true},
		{"最短长度", "abc", true},
		{"最长长度", strings.Repeat("a", 32), true},
		{"过短", "ab", false},
		{"过长", strings.Repeat("a", 33), false},
		{"含空格", "my user", false},
		{"含斜杠（会在 URL 路径中产生歧义）", "my/user", false},
		{"含点", "my.user", false},
		{"中文（避免路径与日志歧义）", "管理员", false},
		{"含 emoji", "user😀", false},
		{"空", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateUsername(c.in)
			if c.valid && err != nil {
				t.Errorf("ValidateUsername(%q) 期望通过，实际报错: %v", c.in, err)
			}
			if !c.valid && err == nil {
				t.Errorf("ValidateUsername(%q) 期望报错，实际通过", c.in)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		valid bool
	}{
		{"足够强的密码", "Str0ngPassw0rd!", true},
		{"恰好 8 字符", "aB3!xY9z", true},
		{"过长（bcrypt 72 字节上限）", strings.Repeat("a", 73), false},
		{"过短", "short1", false},
		{"纯数字弱口令", "12345678", false},
		{"包含 password", "mypassword123", false},
		{"包含 admin", "admin12345", false},
		{"包含项目名 boxli", "boxli2026!", false},
		{"大小写变体也应被识别", "PassWord123", false},
		{"空", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidatePassword(c.in)
			if c.valid && err != nil {
				t.Errorf("ValidatePassword(%q) 期望通过，实际报错: %v", c.in, err)
			}
			if !c.valid && err == nil {
				t.Errorf("ValidatePassword(%q) 期望报错，实际通过", c.in)
			}
		})
	}
}

// 密码强度错误必须是 ErrWeakPassword，便于上层用 errors.Is 分类处理。
func TestValidatePasswordErrorType(t *testing.T) {
	err := ValidatePassword("short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Errorf("期望 ErrWeakPassword，实际: %v", err)
	}
	err = ValidateUsername("a")
	if !errors.Is(err, ErrInvalidUsername) {
		t.Errorf("期望 ErrInvalidUsername，实际: %v", err)
	}
}
