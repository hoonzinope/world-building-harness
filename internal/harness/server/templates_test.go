package server

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonzi/world-harness/internal/harness/ui"
)

func TestAdminUsersTemplateWiresCSRF(t *testing.T) {
	srv := &webServer{}
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	users := []map[string]any{
		{
			"username":        "alice",
			"display_name":    "Alice",
			"role":            "friend",
			"status":          "active",
			"last_login_at":   "",
			"active_sessions": 2,
			"id":              "user_alice",
		},
	}
	srv.render(rec, req, "Admin Users", ui.AdminUsersTemplate, map[string]any{
		"Base":      "",
		"User":      &authUser{ID: "user_admin", Role: "admin"},
		"Users":     users,
		"CSRFToken": "csrf-test",
	})
	html := rec.Body.String()
	for _, want := range []string{
		`for="admin-create-username"`,
		`id="admin-create-username"`,
		`for="admin-create-display-name"`,
		`id="admin-create-display-name"`,
		`for="admin-create-role"`,
		`id="admin-create-role"`,
		`for="admin-create-password"`,
		`id="admin-create-password"`,
		`name="csrf_token" value="csrf-test"`,
		`name="action" value="create"`,
		`name="action" value="update"`,
		`name="action" value="reset"`,
		`name="action" value="revoke"`,
		`aria-label="alice role"`,
		`aria-label="alice status"`,
		`aria-label="alice new password"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in admin users template", want)
		}
	}
}

func TestLayoutTemplateProvidesSkipLinkAndMainLandmark(t *testing.T) {
	srv := &webServer{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.render(rec, req, "World Packs", ui.IndexTemplate, map[string]any{
		"Base":  "",
		"Packs": []any{},
	})
	html := rec.Body.String()
	for _, want := range []string{
		`<a class="skip-link" href="#main-content">본문으로 건너뛰기</a>`,
		`<main class="shell" id="main-content" tabindex="-1">`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in layout", want)
		}
	}
}

func TestPackAndDocTemplatesUseAccessibleNavigationAndSearch(t *testing.T) {
	srv := &webServer{}
	req := httptest.NewRequest(http.MethodGet, "/packs/lumen-federation", nil)
	rec := httptest.NewRecorder()
	srv.render(rec, req, "Lumen Federation", ui.PackTemplate, map[string]any{
		"Base":    "",
		"Title":   "Lumen Federation",
		"Query":   "",
		"Summary": map[string]any{"content_documents": 1, "active_drafts": 0},
		"Types":   []string{},
		"Groups":  map[string][]any{},
	})
	html := rec.Body.String()
	for _, want := range []string{
		`<label class="sr-only" for="pack-search">문서 검색</label>`,
		`id="pack-search"`,
		`button type="submit">검색</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in pack template", want)
		}
	}

	docReq := httptest.NewRequest(http.MethodGet, "/packs/lumen-federation/doc", nil)
	docRec := httptest.NewRecorder()
	srv.render(docRec, docReq, "Doc", ui.DocTemplate, map[string]any{
		"Base":     "",
		"Pack":     "lumen-federation",
		"Doc":      map[string]any{"title": "Test", "type": "canon", "status": "draft", "path": "foo.md"},
		"BodyHTML": template.HTML("<p>body</p>"),
	})
	docHTML := docRec.Body.String()
	if !strings.Contains(docHTML, `세계관 목록으로 돌아가기`) {
		t.Fatal("missing localized back link in doc template")
	}
}

func TestLoginTemplateUsesAccessibleFormStructure(t *testing.T) {
	srv := &webServer{}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	srv.render(rec, req, "Login", ui.LoginTemplate, map[string]any{
		"Base":      "",
		"CSRFToken": "csrf-test",
		"Error":     "로그인 정보를 확인할 수 없습니다.",
	})
	html := rec.Body.String()
	for _, want := range []string{
		`class="auth-shell"`,
		`class="auth-panel"`,
		`class="auth-form"`,
		`for="login-username"`,
		`id="login-username"`,
		`autocomplete="username"`,
		`for="login-password"`,
		`id="login-password"`,
		`autocomplete="current-password"`,
		`role="alert"`,
		`id="login-error"`,
		`aria-invalid="true"`,
		`aria-describedby="login-error"`,
		`class="primary-button"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in login template", want)
		}
	}
	if !strings.Contains(html, `name="csrf_token" value="csrf-test"`) {
		t.Fatal("missing csrf token in login template")
	}
}

func TestStoryLobbyTemplateKeepsRefinedCardAndFilterStructure(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사"); err != nil {
		t.Fatal(err)
	}

	html := renderStoryLobbyHTML(t, &webServer{stories: store}, nil, "?filter=active")
	for _, want := range []string{
		`class="story-lobby-shell"`,
		`class="story-lobby-header"`,
		`class="story-lobby-note"`,
		`story-lobby-filters`,
		`is-selected`,
		`story-card-head`,
		`story-card-foot`,
		`story-card-meta`,
		`story-card-summary`,
		`story-card-actions`,
		`입장하기`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in story lobby template", want)
		}
	}
	for _, forbidden := range []string{
		`story-lobby-table`,
		`<table`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("unexpected %q in story lobby template", forbidden)
		}
	}
}
