package hub

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("repository already exists")
	ErrInvalid   = errors.New("invalid payload")
)

// SourceInput 提交/更新时的下载源输入。
type SourceInput struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Region   string `json:"region"`
	Digest   string `json:"digest"`
	Size     int64  `json:"size_bytes"`
}

// TagInput 提交/更新时的标签输入。
type TagInput struct {
	Tag       string        `json:"tag"`
	OS        string        `json:"os"`
	Arch      string        `json:"arch"`
	Digest    string        `json:"digest"`
	SizeBytes int64         `json:"size_bytes"`
	Sources   []SourceInput `json:"sources"`
}

// RepoInput 提交/更新仓库的请求体。
type RepoInput struct {
	Namespace   string     `json:"namespace"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Readme      string     `json:"readme"`
	Tags        []TagInput `json:"tags"`
}

func (in *RepoInput) validate() error {
	if in.Namespace == "" || in.Name == "" {
		return fmt.Errorf("%w: namespace and name are required", ErrInvalid)
	}
	if len(in.Name) > 128 || len(in.Namespace) > 64 {
		return fmt.Errorf("%w: namespace/name too long", ErrInvalid)
	}
	if len(in.Tags) == 0 {
		return fmt.Errorf("%w: at least one tag is required", ErrInvalid)
	}
	for _, t := range in.Tags {
		if t.Tag == "" || t.OS == "" || t.Arch == "" {
			return fmt.Errorf("%w: tag, os, arch are required", ErrInvalid)
		}
		if len(t.Sources) == 0 {
			return fmt.Errorf("%w: tag %s needs at least one source", ErrInvalid, t.Tag)
		}
		for _, s := range t.Sources {
			if s.Type == "" || s.URL == "" {
				return fmt.Errorf("%w: source type and url are required", ErrInvalid)
			}
		}
	}
	return nil
}

// createRepo 创建仓库（含标签与源），事务内完成。
func (s *store) createRepo(ctx context.Context, ownerID int64, in RepoInput) (*RepoDetail, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var repoID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO repositories (namespace, name, description, readme, owner_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		in.Namespace, in.Name, in.Description, in.Readme, ownerID).Scan(&repoID)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("insert repo: %w", err)
	}

	if err := insertTagsAndSources(ctx, tx, repoID, in.Tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getRepo(ctx, in.Namespace, in.Name)
}

// updateRepo 更新仓库：仅 owner 可改，标签与源整体替换。
func (s *store) updateRepo(ctx context.Context, ownerID int64, in RepoInput) (*RepoDetail, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var (
		repoID    int64
		repoOwner int64
	)
	err = tx.QueryRow(ctx,
		`SELECT id, owner_id FROM repositories WHERE namespace = $1 AND name = $2`,
		in.Namespace, in.Name).Scan(&repoID, &repoOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if repoOwner != ownerID {
		return nil, ErrForbidden
	}

	if _, err := tx.Exec(ctx, `
		UPDATE repositories
		SET description = $1, readme = $2, updated_at = NOW()
		WHERE id = $3`, in.Description, in.Readme, repoID); err != nil {
		return nil, fmt.Errorf("update repo: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM tags WHERE repo_id = $1`, repoID); err != nil {
		return nil, fmt.Errorf("clear tags: %w", err)
	}
	if err := insertTagsAndSources(ctx, tx, repoID, in.Tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getRepo(ctx, in.Namespace, in.Name)
}

// deleteRepo 删除仓库：仅 owner 可删（tags/sources 由外键级联删除）。
func (s *store) deleteRepo(ctx context.Context, ownerID int64, ns, name string) error {
	var repoOwner int64
	err := s.pool.QueryRow(ctx,
		`SELECT owner_id FROM repositories WHERE namespace = $1 AND name = $2`,
		ns, name).Scan(&repoOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if repoOwner != ownerID {
		return ErrForbidden
	}
	_, err = s.pool.Exec(ctx,
		`DELETE FROM repositories WHERE namespace = $1 AND name = $2`, ns, name)
	return err
}

func insertTagsAndSources(ctx context.Context, tx pgx.Tx, repoID int64, tags []TagInput) error {
	for _, t := range tags {
		var tagID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO tags (repo_id, tag, os, arch, digest, size_bytes)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, 0))
			RETURNING id`,
			repoID, t.Tag, t.OS, t.Arch, t.Digest, t.SizeBytes).Scan(&tagID); err != nil {
			return fmt.Errorf("insert tag %s: %w", t.Tag, err)
		}
		for _, src := range t.Sources {
			if _, err := tx.Exec(ctx, `
				INSERT INTO sources (tag_id, type, url, priority, region, digest, size_bytes)
				VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, 0))`,
				tagID, src.Type, src.URL, src.Priority, src.Region, src.Digest, src.Size); err != nil {
				return fmt.Errorf("insert source %s: %w", src.URL, err)
			}
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
