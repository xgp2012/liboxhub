-- 本地密码认证与管理员标记。
--
-- 背景：首个版本只有 GitHub OAuth，没有密码字段。为支持「首次部署引导创建
-- 管理员账号」，需要：
--   1. password_hash —— 本地密码登录（bcrypt）
--   2. is_admin      —— 首个账号标记为管理员
--
-- 两者都可空/有默认值，因此对既有数据完全兼容：已有 OAuth 用户
-- password_hash 为 NULL，依旧只能走 GitHub 登录。

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

-- 用户名需要支持大小写不敏感的唯一性判断（登录时用户可能输入任意大小写）。
-- 现有 UNIQUE(username) 是大小写敏感的，这里补一个不敏感的唯一索引，
-- 避免出现 "Alice" 与 "alice" 两个账号导致的混淆与冒用风险。
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower
    ON users (LOWER(username));
