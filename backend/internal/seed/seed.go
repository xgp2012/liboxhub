package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type source struct {
	Type     string
	URL      string
	Priority int
	Region   string
}

type tag struct {
	Tag       string
	OS        string
	Arch      string
	SizeBytes int64
	Sources   []source
}

type repo struct {
	Namespace   string
	Owner       string
	Email       string
	Name        string
	Description string
	Readme      string
	Stars       int
	Pulls       int64
	Tags        []tag
}

var data = []repo{
	{
		Namespace: "alice", Owner: "alice", Email: "alice@example.com",
		Name: "myapp", Description: "A tiny demo web app packaged as a .boxli image",
		Readme: "# myapp\n\nA tiny demo web app.\n\n```\nboxli pull alice/myapp:v1\nboxli run alice/myapp:v1\n```\n",
		Stars:  128, Pulls: 20450,
		Tags: []tag{
			{Tag: "v1.0.0", OS: "linux", Arch: "amd64", SizeBytes: 5242880, Sources: []source{
				{Type: "github", URL: "https://github.com/alice/myapp/releases/download/v1/myapp.boxli", Priority: 1, Region: "global"},
				{Type: "gitee", URL: "https://gitee.com/alice/myapp/releases/download/v1/myapp.boxli", Priority: 2, Region: "cn"},
				{Type: "oss", URL: "https://alice.oss-cn-hangzhou.aliyuncs.com/myapp-v1.boxli", Priority: 3, Region: "cn"},
			}},
			{Tag: "v1.0.0", OS: "linux", Arch: "arm64", SizeBytes: 5033165, Sources: []source{
				{Type: "github", URL: "https://github.com/alice/myapp/releases/download/v1/myapp-arm64.boxli", Priority: 1, Region: "global"},
			}},
			{Tag: "latest", OS: "linux", Arch: "amd64", SizeBytes: 5242880, Sources: []source{
				{Type: "github", URL: "https://github.com/alice/myapp/releases/download/v1/myapp.boxli", Priority: 1, Region: "global"},
			}},
		},
	},
	{
		Namespace: "bob", Owner: "bob", Email: "bob@example.com",
		Name: "nginx-lite", Description: "Minimal nginx image with a self-signed cert",
		Readme: "# nginx-lite\n\nMinimal nginx for edge deployments.\n",
		Stars:  86, Pulls: 9870,
		Tags: []tag{
			{Tag: "v1.25", OS: "linux", Arch: "amd64", SizeBytes: 12582912, Sources: []source{
				{Type: "github", URL: "https://github.com/bob/nginx-lite/releases/download/v1.25/nginx.boxli", Priority: 1, Region: "global"},
				{Type: "gitee", URL: "https://gitee.com/bob/nginx-lite/releases/download/v1.25/nginx.boxli", Priority: 2, Region: "cn"},
			}},
			{Tag: "v1.25", OS: "linux", Arch: "arm64", SizeBytes: 12058624, Sources: []source{
				{Type: "github", URL: "https://github.com/bob/nginx-lite/releases/download/v1.25/nginx-arm64.boxli", Priority: 1, Region: "global"},
			}},
		},
	},
	{
		Namespace: "carol", Owner: "carol", Email: "carol@example.com",
		Name: "redis-edge", Description: "Redis tuned for ARM edge devices",
		Readme: "# redis-edge\n\nRedis for ARM edge.\n",
		Stars:  210, Pulls: 45200,
		Tags: []tag{
			{Tag: "v7.2", OS: "linux", Arch: "arm64", SizeBytes: 8388608, Sources: []source{
				{Type: "github", URL: "https://github.com/carol/redis-edge/releases/download/v7.2/redis.boxli", Priority: 1, Region: "global"},
				{Type: "ipfs", URL: "ipfs://QmT78zSuBmuS4z925WZfrqQ1qHaJ56DQaTfyMUF7F8ff5o", Priority: 4, Region: "global"},
			}},
			{Tag: "v7.2", OS: "linux", Arch: "amd64", SizeBytes: 8912896, Sources: []source{
				{Type: "s3", URL: "https://s3.amazonaws.com/carol-images/redis-v7.2.boxli", Priority: 2, Region: "us"},
			}},
		},
	},
	{
		Namespace: "dave", Owner: "dave", Email: "dave@example.com",
		Name: "postgres-boxli", Description: "PostgreSQL 16 packaged image",
		Readme: "# postgres-boxli\n\nPostgreSQL 16 ready to run.\n",
		Stars:  342, Pulls: 120300,
		Tags: []tag{
			{Tag: "v16", OS: "linux", Arch: "amd64", SizeBytes: 60817408, Sources: []source{
				{Type: "github", URL: "https://github.com/dave/postgres-boxli/releases/download/v16/pg.boxli", Priority: 1, Region: "global"},
				{Type: "cos", URL: "https://dave-1250000000.cos.ap-guangzhou.myqcloud.com/pg-v16.boxli", Priority: 3, Region: "cn"},
				{Type: "http", URL: "https://mirror.example.com/pg-v16.boxli", Priority: 10, Region: "global"},
			}},
			{Tag: "v16", OS: "linux", Arch: "arm64", SizeBytes: 58720256, Sources: []source{
				{Type: "github", URL: "https://github.com/dave/postgres-boxli/releases/download/v16/pg-arm64.boxli", Priority: 1, Region: "global"},
			}},
		},
	},
	{
		Namespace: "erin", Owner: "erin", Email: "erin@example.com",
		Name: "hello-riscv", Description: "Hello world running on riscv64",
		Readme: "# hello-riscv\n\nProof that boxli runs on riscv64.\n",
		Stars:  57, Pulls: 3210,
		Tags: []tag{
			{Tag: "v0.1", OS: "linux", Arch: "riscv64", SizeBytes: 2097152, Sources: []source{
				{Type: "github", URL: "https://github.com/erin/hello-riscv/releases/download/v0.1/hello.boxli", Priority: 1, Region: "global"},
				{Type: "magnet", URL: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567", Priority: 20, Region: "global"},
			}},
		},
	},
}

// Run 幂等写入种子数据。已存在的仓库跳过。
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	for _, r := range data {
		var ownerID int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (username, email)
			VALUES ($1, $2)
			ON CONFLICT (username) DO UPDATE SET email = EXCLUDED.email
			RETURNING id`, r.Owner, r.Email).Scan(&ownerID); err != nil {
			return fmt.Errorf("user %s: %w", r.Owner, err)
		}

		var repoID int64
		err := pool.QueryRow(ctx, `
			INSERT INTO repositories (namespace, name, description, readme, owner_id, stars, pulls)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (namespace, name) DO NOTHING
			RETURNING id`,
			r.Namespace, r.Name, r.Description, r.Readme, ownerID, r.Stars, r.Pulls).Scan(&repoID)

		if err != nil { // 已存在，跳过该 repo 全部数据
			continue
		}

		for _, t := range r.Tags {
			var tagID int64
			if err := pool.QueryRow(ctx, `
				INSERT INTO tags (repo_id, tag, os, arch, size_bytes)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id`,
				repoID, t.Tag, t.OS, t.Arch, t.SizeBytes).Scan(&tagID); err != nil {
				return fmt.Errorf("tag %s/%s:%s: %w", r.Namespace, r.Name, t.Tag, err)
			}
			for _, s := range t.Sources {
				if _, err := pool.Exec(ctx, `
					INSERT INTO sources (tag_id, type, url, priority, region, size_bytes)
					VALUES ($1, $2, $3, $4, $5, $6)`,
					tagID, s.Type, s.URL, s.Priority, s.Region, t.SizeBytes); err != nil {
					return fmt.Errorf("source %s: %w", s.URL, err)
				}
			}
		}
	}
	return nil
}
