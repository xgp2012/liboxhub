package hub

import (
	"errors"
	"net/http"
	"strings"

	"github.com/LiStudioorg/boxli/internal/auth"
	"github.com/LiStudioorg/boxli/internal/localauth"
)

// setupPageHTML 是首次部署引导页。
//
// 为什么内嵌一段简单 HTML 而不用 Nuxt 页面：
// 引导必须在**前端尚未配置**的情况下可用（此时前端可能还没有正确的
// frontend_url 配置），因此不能依赖 SPA 路由与 API 代理。
// 这里只用一个自包含表单，减少引导失败的可能。
const setupPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>初始化 Boxli Hub</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body { margin: 0; min-height: 100dvh; display: grid; place-items: center;
         background: #09090b; color: #e4e4e7; font-family: system-ui, -apple-system, "Segoe UI", sans-serif; }
  .card { width: 100%; max-width: 26rem; padding: 2rem; }
  h1 { font-size: 1.25rem; margin: 0 0 .5rem; }
  p.sub { margin: 0 0 1.5rem; color: #a1a1aa; font-size: .875rem; line-height: 1.6; }
  label { display: block; font-size: .8125rem; color: #a1a1aa; margin-bottom: .375rem; }
  input { width: 100%; min-height: 2.75rem; padding: 0 .75rem; font-size: 1rem;
          background: #18181b; border: 1px solid #27272a; border-radius: .375rem; color: #e4e4e7; }
  input:focus { outline: none; border-color: #52525b; }
  .field { margin-bottom: 1rem; }
  button { width: 100%; min-height: 2.75rem; font-size: .9375rem; font-weight: 500;
           background: #e4e4e7; color: #18181b; border: 0; border-radius: .375rem; cursor: pointer; }
  button:hover { background: #fff; }
  button:disabled { opacity: .5; cursor: default; }
  .msg { margin-top: 1rem; padding: .75rem; border-radius: .375rem; font-size: .875rem; display: none; }
  .msg.err { display: block; background: #450a0a; border: 1px solid #7f1d1d; color: #fca5a5; }
  .msg.ok  { display: block; background: #052e16; border: 1px solid #14532d; color: #86efac; }
  .hint { margin-top: 1.5rem; font-size: .75rem; color: #71717a; line-height: 1.6; }
  code { background: #27272a; padding: .125rem .375rem; border-radius: .25rem; font-size: .9em; }
</style>
</head>
<body>
<div class="card">
  <h1>初始化 Boxli Hub</h1>
  <p class="sub">这是首次启动。请创建管理员账号，完成后本页面将自动关闭。</p>

  <form id="f" autocomplete="off">
    <div class="field">
      <label for="u">管理员用户名</label>
      <input id="u" name="username" required minlength="3" maxlength="32"
             autocapitalize="none" autocorrect="off" spellcheck="false"
             placeholder="仅限字母、数字、下划线、连字符">
    </div>
    <div class="field">
      <label for="p">密码</label>
      <input id="p" name="password" type="password" required minlength="8"
             autocomplete="new-password" placeholder="至少 8 个字符">
    </div>
    <div class="field">
      <label for="p2">确认密码</label>
      <input id="p2" name="confirm" type="password" required minlength="8"
             autocomplete="new-password">
    </div>
    <button id="b" type="submit">创建管理员并完成初始化</button>
  </form>

  <div id="m" class="msg" role="status" aria-live="polite"></div>

  <p class="hint">
    引导令牌已自动从当前地址读取。初始化完成后，本接口会永久关闭
    （仅当系统中没有任何用户时才可用）。
  </p>
</div>

<script>
(function () {
  var form = document.getElementById('f');
  var msg = document.getElementById('m');
  var btn = document.getElementById('b');

  function show(text, kind) {
    msg.textContent = text;
    msg.className = 'msg ' + kind;
  }

  form.addEventListener('submit', async function (e) {
    e.preventDefault();
    var u = document.getElementById('u').value.trim();
    var p = document.getElementById('p').value;
    var p2 = document.getElementById('p2').value;

    if (p !== p2) { show('两次输入的密码不一致', 'err'); return; }

    btn.disabled = true;
    try {
      var token = new URLSearchParams(location.search).get('token') || '';
      var res = await fetch('/api/v1/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        // 令牌同时放在 body 与请求头，便于后端一致地校验
        body: JSON.stringify({ token: token, username: u, password: p })
      });
      var data = await res.json();
      if (data.code === 0) {
        show('初始化完成，正在跳转到登录页…', 'ok');
        setTimeout(function () { location.href = '/login'; }, 1200);
      } else {
        show(data.message || '初始化失败', 'err');
        btn.disabled = false;
      }
    } catch (err) {
      show('请求失败：' + err.message, 'err');
      btn.disabled = false;
    }
  });
})();
</script>
</body>
</html>`

// handleSetupPage GET /setup
// 返回自包含的引导页面。仅在系统尚无用户时可用。
func (s *Server) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 令牌通过查询串传给页面（页面再用它调用 POST /api/v1/setup）。
	// 这里不做校验：页面本身不含敏感信息，真正把门的是 POST 接口。
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(setupPageHTML))
}

// handleSetup POST /api/v1/setup
//
// 首次部署引导：创建管理员账号。
//
// 三重防护（缺一不可）：
//  1. 仅在 users 表为空时允许 —— 一旦有人注册，接口永久失效
//  2. 必须携带一次性安装令牌 —— 令牌在启动时随机生成并打印到**服务端日志**，
//     公网访问者看不到日志，因此无法抢先注册
//  3. 令牌比较用 constant-time，避免时序侧信道
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.local == nil {
		writeErr(w, http.StatusServiceUnavailable, "本地认证未启用")
		return
	}

	var body struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	// ---- 防护 1：令牌校验（最先执行，避免无令牌请求触及数据库）----
	if !s.setupTokenOK(body.Token) {
		writeErr(w, http.StatusForbidden,
			"安装令牌无效。请查看服务端启动日志中打印的 setup token。")
		return
	}

	// ---- 防护 2：仅在系统无用户时允许 ----
	// 计数与插入在同一事务内完成（见 CreateFirstAdmin），
	// 避免并发请求同时通过检查而创建多个管理员。
	created, err := s.local.CreateFirstAdmin(r.Context(), body.Username, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, localauth.ErrInvalidUsername),
			errors.Is(err, localauth.ErrWeakPassword):
			// 这类错误是给用户看的，直接回显
			writeErr(w, http.StatusBadRequest, err.Error())
		default:
			writeErr(w, http.StatusInternalServerError, "创建管理员失败")
		}
		return
	}
	if !created {
		writeErr(w, http.StatusConflict,
			"系统已存在用户，初始化引导已关闭。请直接登录。")
		return
	}

	// 初始化完成：令牌立即作废，避免被重复利用。
	s.invalidateSetupToken()

	writeOK(w, map[string]any{
		"initialized": true,
		"username":    strings.TrimSpace(body.Username),
	})
}

// setupTokenOK 以恒定时间比较安装令牌。
func (s *Server) setupTokenOK(got string) bool {
	s.setupMu.Lock()
	token := s.setupToken
	s.setupMu.Unlock()

	if token == "" {
		return false // 已初始化或未启用引导
	}
	return auth.ConstantTimeEqual(token, got)
}

// invalidateSetupToken 作废安装令牌。
func (s *Server) invalidateSetupToken() {
	s.setupMu.Lock()
	s.setupToken = ""
	s.setupMu.Unlock()
}
