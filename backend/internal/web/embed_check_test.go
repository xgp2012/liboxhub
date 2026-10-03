package web

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEmbedContainsNuxtAssets 守住 go:embed 的 all: 前缀。
// 若有人把 `//go:embed all:dist` 改回 `//go:embed dist`，
// _nuxt 目录会被静默丢弃，本测试会失败。
func TestEmbedContainsNuxtAssets(t *testing.T) {
	sub, err := Assets()
	if err != nil {
		t.Fatalf("Assets(): %v", err)
	}
	var count int
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if count == 0 {
		t.Fatal("内嵌静态资源为空：go:embed 前缀可能缺少 all:（点/下划线目录被忽略）")
	}
	t.Logf("内嵌静态资源文件数: %d", count)

	// 必须存在 _nuxt 目录，这是最容易因 embed 前缀写错而丢失的部分
	found := false
	_ = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && strings.HasPrefix(p, "_nuxt") {
			found = true
		}
		return nil
	})
	if !found {
		t.Error("未找到 _nuxt 目录：go:embed 很可能缺少 all: 前缀")
	}
}

func TestSSRBundleEmbedded(t *testing.T) {
	b, err := SSRBundle()
	if err != nil {
		t.Fatalf("SSRBundle(): %v", err)
	}
	if len(b) < 1<<20 {
		t.Fatalf("SSR bundle 过小（%d 字节），可能未正确打包", len(b))
	}
	t.Logf("内嵌 SSR bundle: %.1f MB", float64(len(b))/1048576)
}
