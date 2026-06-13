package elife_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/elife-cli/elife"
)

func newTestClient(ts *httptest.Server) *elife.Client {
	cfg := elife.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return elife.NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"items":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, _, err := c.Recent(context.Background(), elife.ListOptions{PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSendsAcceptHeader(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"items":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, _, err := c.Recent(context.Background(), elife.ListOptions{PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotAccept, "article-list") {
		t.Errorf("Accept header %q does not contain article-list", gotAccept)
	}
}

func TestRetryOn503(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":0,"items":[]}`))
	}))
	defer srv.Close()

	cfg := elife.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := elife.NewClient(cfg)

	_, _, err := c.Recent(context.Background(), elife.ListOptions{PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestRecentReturnsArticles(t *testing.T) {
	body := `{
		"total": 2,
		"items": [
			{
				"id": "85111",
				"title": "Test Article One",
				"published": "2023-08-01",
				"statusDate": "2023-08-01",
				"type": "research-article",
				"status": "vor",
				"subjects": [{"id": "cell-biology", "name": "Cell Biology"}],
				"doi": "10.7554/eLife.85111"
			},
			{
				"id": "85222",
				"title": "Test Article Two",
				"published": "2023-07-15",
				"statusDate": "2023-07-15",
				"type": "editorial",
				"status": "vor",
				"subjects": [],
				"doi": "10.7554/eLife.85222"
			}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	items, total, err := c.Recent(context.Background(), elife.ListOptions{PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	a := items[0]
	if a.ID != "85111" {
		t.Errorf("ID = %q, want 85111", a.ID)
	}
	if a.Title != "Test Article One" {
		t.Errorf("Title = %q", a.Title)
	}
	if a.Published != "2023-08-01" {
		t.Errorf("Published = %q", a.Published)
	}
	if a.DOI != "10.7554/eLife.85111" {
		t.Errorf("DOI = %q", a.DOI)
	}
	if a.URL != "https://elifesciences.org/articles/85111" {
		t.Errorf("URL = %q", a.URL)
	}
	if a.Subjects != "Cell Biology" {
		t.Errorf("Subjects = %q", a.Subjects)
	}
	if a.Rank != 1 {
		t.Errorf("Rank = %d, want 1", a.Rank)
	}
	if items[1].Rank != 2 {
		t.Errorf("second item rank = %d, want 2", items[1].Rank)
	}
}

func TestSearchReturnsArticles(t *testing.T) {
	body := `{
		"total": 1,
		"items": [
			{
				"id": "99999",
				"title": "CRISPR in Cell Biology",
				"published": "2024-01-01",
				"statusDate": "2024-01-01",
				"type": "research-article",
				"status": "vor",
				"subjects": [{"id": "cell-biology", "name": "Cell Biology"}, {"id": "genetics-genomics", "name": "Genetics and Genomics"}],
				"doi": "10.7554/eLife.99999"
			}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	items, total, err := c.Search(context.Background(), elife.SearchOptions{Query: "CRISPR", PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	a := items[0]
	if a.Rank != 1 {
		t.Errorf("Rank = %d, want 1", a.Rank)
	}
	if a.ID != "99999" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.Subjects != "Cell Biology; Genetics and Genomics" {
		t.Errorf("Subjects = %q", a.Subjects)
	}
}

func TestArticleReturnsFullRecord(t *testing.T) {
	body := `{
		"id": "85111",
		"title": "Full Article Title",
		"abstract": {
			"content": [
				{"text": "This is the abstract."},
				{"text": "Second paragraph."}
			]
		},
		"published": "2023-08-01",
		"statusDate": "2023-08-01",
		"volume": 12,
		"issue": 1,
		"elocationId": "e85111",
		"type": "research-article",
		"status": "vor",
		"doi": "10.7554/eLife.85111",
		"subjects": [{"id": "cell-biology", "name": "Cell Biology"}],
		"authors": [
			{"type": "person", "name": {"given": "Jane", "surname": "Doe"}},
			{"type": "group", "name": "eLife Team"}
		],
		"copyright": {
			"license": "CC-BY-4.0",
			"holder": "Doe et al.",
			"statement": "..."
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	a, err := c.Article(context.Background(), "85111")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "85111" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.Title != "Full Article Title" {
		t.Errorf("Title = %q", a.Title)
	}
	if a.Abstract != "This is the abstract. Second paragraph." {
		t.Errorf("Abstract = %q", a.Abstract)
	}
	if a.Volume != 12 {
		t.Errorf("Volume = %d", a.Volume)
	}
	if a.ElocationID != "e85111" {
		t.Errorf("ElocationID = %q", a.ElocationID)
	}
	if a.Authors != "Jane Doe; eLife Team" {
		t.Errorf("Authors = %q", a.Authors)
	}
	if a.License != "CC-BY-4.0" {
		t.Errorf("License = %q", a.License)
	}
	if a.URL != "https://elifesciences.org/articles/85111" {
		t.Errorf("URL = %q", a.URL)
	}
}

func TestSubjectsReturnsList(t *testing.T) {
	body := `{
		"total": 2,
		"items": [
			{"id": "biochemistry-chemical-biology", "name": "Biochemistry and Chemical Biology"},
			{"id": "cell-biology", "name": "Cell Biology"}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	subjects, err := c.Subjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(subjects) != 2 {
		t.Fatalf("len(subjects) = %d, want 2", len(subjects))
	}
	if subjects[0].Rank != 1 {
		t.Errorf("first subject rank = %d, want 1", subjects[0].Rank)
	}
	if subjects[0].ID != "biochemistry-chemical-biology" {
		t.Errorf("first subject ID = %q", subjects[0].ID)
	}
	if subjects[0].Name != "Biochemistry and Chemical Biology" {
		t.Errorf("first subject Name = %q", subjects[0].Name)
	}
	if subjects[1].Rank != 2 {
		t.Errorf("second subject rank = %d, want 2", subjects[1].Rank)
	}
}

func TestNotFoundReturns404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Article(context.Background(), "00000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got %q", err.Error())
	}
}

func TestArticleURL(t *testing.T) {
	body := `{"total":1,"items":[{"id":"12345","title":"T","published":"2024","statusDate":"2024","type":"editorial","status":"vor","subjects":[],"doi":"10.7554/eLife.12345"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	items, _, err := c.Recent(context.Background(), elife.ListOptions{PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("no items")
	}
	want := "https://elifesciences.org/articles/12345"
	if items[0].URL != want {
		t.Errorf("URL = %q, want %q", items[0].URL, want)
	}
}

func TestJoinSubjectsEmpty(t *testing.T) {
	body := `{"total":1,"items":[{"id":"1","title":"T","published":"2024","statusDate":"2024","type":"research-article","status":"vor","subjects":[],"doi":"10.7554/eLife.1"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	items, _, err := c.Recent(context.Background(), elife.ListOptions{PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Subjects != "" {
		t.Errorf("expected empty subjects, got %q", items[0].Subjects)
	}
}

func TestFormatAuthors(t *testing.T) {
	body := `{
		"id": "99",
		"title": "T",
		"published": "2024",
		"statusDate": "2024",
		"type": "research-article",
		"status": "vor",
		"doi": "10.7554/eLife.99",
		"subjects": [],
		"authors": [
			{"type": "person", "name": {"given": "Alice", "surname": "Smith"}},
			{"type": "person", "name": {"given": "Bob", "surname": "Jones"}},
			{"type": "group", "name": "The Consortium"}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	a, err := c.Article(context.Background(), "99")
	if err != nil {
		t.Fatal(err)
	}
	want := "Alice Smith; Bob Jones; The Consortium"
	if a.Authors != want {
		t.Errorf("Authors = %q, want %q", a.Authors, want)
	}
}
