package search

import (
	"context"
)

type IndexableDocument struct {
	ID      string
	Content string
	Title   string
	SiteID  string
	Type    string
}

type SearchResult struct {
	ID      string
	Score   float64
	Title   string
	Excerpt string
	Type    string
}

type Engine interface {
	Index(ctx context.Context, doc IndexableDocument) error
	Search(ctx context.Context, siteID, query string, limit, offset int) ([]SearchResult, int, error)
	Delete(ctx context.Context, id string) error
	Reindex(ctx context.Context, docs []IndexableDocument) error
}
