package auth

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestConstantTimeEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"相同", "abc123", "abc123", true},
		{"不同", "abc123", "abc124", false},
		{"长度不同", "abc", "abcd", false},
		{"空对空", "", "", true},
		{"空对非空", "", "a", false},
		{"大小写敏感", "ABC", "abc", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ConstantTimeEqual(c.a, c.b); got != c.want {
				t.Errorf("ConstantTimeEqual(%q,%q) = %v, 期望 %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestNewSetupToken(t *testing.T) {
	tok, err := NewSetupToken()
	if err != nil {
		t.Fatalf("NewSetupToken: %v", err)
	}

	// 32 字节 → 64 个十六进制字符
	if len(tok) != 64 {
		t.Errorf("令牌长度 = %d，期望 64", len(tok))
	}
	if _, err := hex.DecodeString(tok); err != nil {
		t.Errorf("令牌不是合法十六进制: %v", err)
	}

	// 两次生成必须不同，否则说明熵源有问题
	tok2, err := NewSetupToken()
	if err != nil {
		t.Fatalf("NewSetupToken 第二次: %v", err)
	}
	if tok == tok2 {
		t.Error("两次生成的令牌相同，随机源可能失效")
	}

	// 不应包含容易被 shell/URL 误解的字符
	if strings.ContainsAny(tok, "+/= ") {
		t.Errorf("令牌含非十六进制字符: %q", tok)
	}
}
