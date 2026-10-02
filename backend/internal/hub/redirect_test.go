package hub

import "testing"

// TestSafeRedirectPath 覆盖开放重定向（open redirect）攻击面：
// 登录后的回跳地址来自前端，绝不能被利用来把用户弹到外部站点。
func TestSafeRedirectPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
		why  string
	}{
		{"/submit", "/submit", "站内相对路径应放行"},
		{"/dashboard", "/dashboard", "站内相对路径应放行"},
		{"/explore/alice/myapp", "/explore/alice/myapp", "多级路径应放行"},
		{"/submit?from=login", "/submit?from=login", "带查询串应放行"},
		{"", "", "空值返回空（调用方回退默认页）"},

		{"//evil.com", "", "协议相对 URL 必须拒绝"},
		{"https://evil.com", "", "绝对 URL 必须拒绝"},
		{"http://evil.com/x", "", "绝对 URL 必须拒绝"},
		{"/\\evil.com", "", "反斜杠变体必须拒绝"},
		{"/path\nSet-Cookie: x=1", "", "控制字符（响应头注入）必须拒绝"},
		{"/path\r\nLocation: https://evil.com", "", "CRLF 必须拒绝"},
		{"javascript:alert(1)", "", "非路径开头必须拒绝"},
		{"submit", "", "缺少前导斜杠必须拒绝"},
	}

	for _, c := range cases {
		if got := safeRedirectPath(c.in); got != c.want {
			t.Errorf("safeRedirectPath(%q) = %q, want %q（%s）", c.in, got, c.want, c.why)
		}
	}
}
