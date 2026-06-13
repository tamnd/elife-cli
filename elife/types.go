package elife

import (
	"encoding/json"
	"strings"
)

// ArticleItem is the record emitted for article lists and search results.
type ArticleItem struct {
	Rank       int    `json:"rank"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Published  string `json:"published"`
	StatusDate string `json:"status_date"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Subjects   string `json:"subjects"`
	DOI        string `json:"doi"`
	URL        string `json:"url"`
}

// Article is the full record for a single article.
type Article struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Abstract    string `json:"abstract"`
	Published   string `json:"published"`
	StatusDate  string `json:"status_date"`
	Volume      int    `json:"volume"`
	Issue       int    `json:"issue"`
	ElocationID string `json:"elocation_id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	DOI         string `json:"doi"`
	Subjects    string `json:"subjects"`
	Authors     string `json:"authors"`
	License     string `json:"license"`
	URL         string `json:"url"`
}

// Subject is the record for a subject area.
type Subject struct {
	Rank int    `json:"rank"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListOptions controls the /articles endpoint.
type ListOptions struct {
	PerPage int
	Page    int
	Order   string // "asc" or "desc"
	Subject string // subject id filter, empty = all
	Type    string // article type filter, empty = all
}

// SearchOptions controls the /search endpoint.
type SearchOptions struct {
	Query   string
	PerPage int
	Page    int
	Order   string
}

// ─── wire types ───────────────────────────────────────────────────────────────

type articleListResp struct {
	Total   int           `json:"total"`
	Items   []articleWire `json:"items"`
	PerPage int           `json:"per-page"`
	Page    int           `json:"page"`
	Pages   int           `json:"pages"`
}

type articleWire struct {
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	Published  string        `json:"published"`
	StatusDate string        `json:"statusDate"`
	Type       string        `json:"type"`
	Status     string        `json:"status"`
	Subjects   []subjectWire `json:"subjects"`
	DOI        string        `json:"doi"`
}

type articleDetailWire struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Abstract    *abstractWire  `json:"abstract"`
	Published   string         `json:"published"`
	StatusDate  string         `json:"statusDate"`
	Volume      int            `json:"volume"`
	Issue       int            `json:"issue"`
	ElocationID string         `json:"elocationId"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	DOI         string         `json:"doi"`
	Subjects    []subjectWire  `json:"subjects"`
	Authors     []authorWire   `json:"authors"`
	Copyright   *copyrightWire `json:"copyright"`
}

type subjectWire struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type abstractWire struct {
	Content []abstractBlock `json:"content"`
}

type abstractBlock struct {
	Text string `json:"text"`
}

type authorWire struct {
	Type string          `json:"type"`
	Name json.RawMessage `json:"name"`
}

type authorName struct {
	Given   string `json:"given"`
	Surname string `json:"surname"`
}

type copyrightWire struct {
	License   string `json:"license"`
	Holder    string `json:"holder"`
	Statement string `json:"statement"`
}

type subjectListResp struct {
	Total int           `json:"total"`
	Items []subjectWire `json:"items"`
}

// ─── conversion helpers ───────────────────────────────────────────────────────

func articleURL(id string) string {
	return "https://elifesciences.org/articles/" + id
}

func joinSubjects(ss []subjectWire) string {
	names := make([]string, 0, len(ss))
	for _, s := range ss {
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, "; ")
}

func formatAuthors(authors []authorWire) string {
	parts := make([]string, 0, len(authors))
	for _, a := range authors {
		parts = append(parts, formatAuthor(a))
	}
	return strings.Join(parts, "; ")
}

func formatAuthor(a authorWire) string {
	if len(a.Name) == 0 {
		return a.Type
	}
	// Try to decode as object first (person with given/surname).
	var n authorName
	if err := json.Unmarshal(a.Name, &n); err == nil && (n.Given != "" || n.Surname != "") {
		full := strings.TrimSpace(n.Given + " " + n.Surname)
		if full != "" {
			return full
		}
	}
	// Try as plain string (group authors).
	var s string
	if err := json.Unmarshal(a.Name, &s); err == nil && s != "" {
		return s
	}
	return a.Type
}

func extractAbstract(a *abstractWire) string {
	if a == nil {
		return ""
	}
	parts := make([]string, 0, len(a.Content))
	for _, b := range a.Content {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, " ")
}

func wireToArticleItem(w articleWire, rank int) ArticleItem {
	return ArticleItem{
		Rank:       rank,
		ID:         w.ID,
		Title:      w.Title,
		Published:  w.Published,
		StatusDate: w.StatusDate,
		Type:       w.Type,
		Status:     w.Status,
		Subjects:   joinSubjects(w.Subjects),
		DOI:        w.DOI,
		URL:        articleURL(w.ID),
	}
}

func wireToArticle(w articleDetailWire) Article {
	license := ""
	if w.Copyright != nil {
		license = w.Copyright.License
	}
	return Article{
		ID:          w.ID,
		Title:       w.Title,
		Abstract:    extractAbstract(w.Abstract),
		Published:   w.Published,
		StatusDate:  w.StatusDate,
		Volume:      w.Volume,
		Issue:       w.Issue,
		ElocationID: w.ElocationID,
		Type:        w.Type,
		Status:      w.Status,
		DOI:         w.DOI,
		Subjects:    joinSubjects(w.Subjects),
		Authors:     formatAuthors(w.Authors),
		License:     license,
		URL:         articleURL(w.ID),
	}
}
