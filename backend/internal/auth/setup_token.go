package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// ConstantTimeEqual 以恒定时间比较两个字符串。
//
// 用于比较安装令牌等机密值：普通的 == 会在首个不同字节处提前返回，
// 攻击者可据此逐字节推断出正确值（时序侧信道）。
//
// 长度不同时直接返回 false —— 长度本身不是机密（令牌长度固定），
// 且 subtle.ConstantTimeCompare 要求等长输入。
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// NewSetupToken 生成首次部署引导用的一次性令牌。
//
// 32 字节随机熵（256 位），以十六进制表示共 64 字符。
// 该强度下暴力猜测不可行，即使引导接口暴露在公网也是安全的。
func NewSetupToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成安装令牌: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
