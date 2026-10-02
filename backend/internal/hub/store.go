package hub

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("repository not found")

// Source 是某个标签的一个下载源。
type Source struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Region   string `json:"region,omitempty"`
	Digest   string `json:"digest,omitempty"`
	Size     int64  `json:"size_bytes,omitempty"`
}

// Tag 是仓库的一个 (tag, os, arch) 组合。
type Tag struct {
	ID        int64    `json:"id"`
	Tag       string   `json:"tag"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	Digest    string   `json:"digest,omitempty"`
	SizeBytes int64    `json:"size_bytes,omitempty"`
	Sources   []Source `json:"sources"`
}

// Repo 是仓库摘要（列表用）。
type Repo struct {
	ID          int64     `json:"id"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Stars       int       `json:"stars"`
	Pulls       int64     `json:"pulls"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RepoDetail 是仓库详情（含标签与源）。
type RepoDetail struct {
	Repo
	Readme    string    `json:"readme"`
	IsPublic  bool      `json:"is_public"`
	OwnerID   int64     `json:"owner_id"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []Tag     `json:"tags"`
}

type store struct {
	pool *pgxpool.Pool
}

// search 按关键词搜索公开仓库，按星标降序。
func (s *store) search(ctx context.Context, q string, limit int) ([]Repo, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, namespace, name, COALESCE(description, ''), stars, pulls, updated_at
		FROM repositories
		WHERE is_public = TRUE
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%'
		       OR namespace ILIKE '%' || $1 || '%')
		ORDER BY stars DESC, pulls DESC
		LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRepos(rows)
}

// listRepos 列出公开仓库，支持按 namespace 过滤。
func (s *store) listRepos(ctx context.Context, ns string, limit, offset int) ([]Repo, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, namespace, name, COALESCE(description, ''), stars, pulls, updated_at
		FROM repositories
		WHERE is_public = TRUE AND ($1 = '' OR namespace = $1)
		ORDER BY stars DESC, pulls DESC
		LIMIT $2 OFFSET $3`, ns, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRepos(rows)
}

func scanRepos(rows pgx.Rows) ([]Repo, error) {
	repos := make([]Repo, 0)
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.Namespace, &r.Name, &r.Description, &r.Stars, &r.Pulls, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// getRepo 拉取仓库详情（含所有标签与源）。
func (s *store) getRepo(ctx context.Context, ns, name string) (*RepoDetail, error) {
	var d RepoDetail
	err := s.pool.QueryRow(ctx, `
		SELECT r.id, r.namespace, r.name, COALESCE(r.description, ''), r.stars, r.pulls,
		       r.updated_at, r.created_at, COALESCE(r.readme, ''), r.is_public, r.owner_id,
		       u.username
		FROM repositories r
		JOIN users u ON u.id = r.owner_id
		WHERE r.namespace = $1 AND r.name = $2 AND r.is_public = TRUE`,
		ns, name,
	).Scan(&d.ID, &d.Namespace, &d.Name, &d.Description, &d.Stars, &d.Pulls,
		&d.UpdatedAt, &d.CreatedAt, &d.Readme, &d.IsPublic, &d.OwnerID, &d.Author)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	d.Tags, err = s.tagsForRepo(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// tagsForRepo 查询仓库所有标签及其源。
func (s *store) tagsForRepo(ctx context.Context, repoID int64) ([]Tag, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tag, os, arch, COALESCE(digest, ''), COALESCE(size_bytes, 0)
		FROM tags WHERE repo_id = $1
		ORDER BY created_at DESC, tag`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]Tag, 0)
	index := map[int64]int{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Tag, &t.OS, &t.Arch, &t.Digest, &t.SizeBytes); err != nil {
			return nil, err
		}
		t.Sources = make([]Source, 0)
		index[t.ID] = len(tags)
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return tags, nil
	}

	srcRows, err := s.pool.Query(ctx, `
		SELECT id, tag_id, type, url, priority, COALESCE(region, ''),
		       COALESCE(digest, ''), COALESCE(size_bytes, 0)
		FROM sources
		WHERE tag_id IN (SELECT id FROM tags WHERE repo_id = $1)
		ORDER BY priority, id`, repoID)
	if err != nil {
		return nil, err
	}
	defer srcRows.Close()
	for srcRows.Next() {
		var (
			src   Source
			tagID int64
		)
		if err := srcRows.Scan(&src.ID, &tagID, &src.Type, &src.URL, &src.Priority, &src.Region, &src.Digest, &src.Size); err != nil {
			return nil, err
		}
		if i, ok := index[tagID]; ok {
			tags[i].Sources = append(tags[i].Sources, src)
		}
	}
	return tags, srcRows.Err()
}

// readme 只取 README 文本。
func (s *store) readme(ctx context.Context, ns, name string) (string, error) {
	var readme string
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(readme, '') FROM repositories
		WHERE namespace = $1 AND name = $2 AND is_public = TRUE`, ns, name).Scan(&readme)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return readme, err
}
