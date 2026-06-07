package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoryRoomProcessingStateOmitsMetaRefreshAndShowsBusyProgress(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createStoryWithPrologueJob("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store}
	html := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if strings.Contains(html, `http-equiv="refresh"`) {
		t.Fatalf("unexpected meta refresh in processing story room")
	}
	for _, want := range []string{
		`id="story-room"`,
		`data-story-input-panel`,
		`aria-busy="true"`,
		`id="story-progress" role="status" aria-live="polite" aria-atomic="true" aria-busy="true"`,
		`data-story-progress`,
		`data-story-progress-meta hidden`,
		`story-progress-steps`,
		`data-story-step="queued"`,
		`data-story-step="generating"`,
		`data-story-step="applying"`,
		`data-story-step="ready"`,
		`대기열에 들어갔습니다. 순서를 기다립니다.`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in processing story room", want)
		}
	}
	if strings.Contains(html, `active job:`) {
		t.Fatalf("unexpected active job copy in processing story room")
	}
}

func TestStoryRoomAssetRouteServed(t *testing.T) {
	srv := &webServer{}
	req := httptest.NewRequest(http.MethodGet, "/assets/story-room.js", nil)
	rec := httptest.NewRecorder()
	srv.handle(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/javascript") {
		t.Fatalf("unexpected content type %q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`async function submitForm(form, submitter)`,
		`submitForm(form, event.submitter || null);`,
		`data-story-submit`,
		`WeakMap`,
		`captureInitialControlState`,
		`restoreInitialControlState`,
		`initialControlState`,
		`control.disabled = initial.disabled`,
		`ariaDisabled`,
		`pollStatus`,
		`inputPanel.setAttribute('aria-busy'`,
		`data-story-progress-meta`,
		`const metaNode = progress.querySelector('[data-story-progress-meta]');`,
		`metaNode.hidden = !visible;`,
		`const actionURL = new URL(form.action, window.location.href);`,
		`const requestURL = actionURL.origin === window.location.origin ? actionURL.pathname + actionURL.search : form.action;`,
		`Object.fromEntries(data.entries())`,
		`requestPayload`,
		`responsePayload`,
		`'Content-Type': 'application/json'`,
		`JSON.stringify(requestPayload)`,
		`credentials: 'include'`,
		`'X-CSRF-Token': form.querySelector('input[name="csrf_token"]')?.value || ''`,
		`progress.dataset.stepIndex = '0';`,
		`progress.dataset.stepLabel = 'queued';`,
		`statusLabel.textContent = friendlyStepLabel('queued');`,
		`setStep('queued');`,
		`제출 응답을 JSON으로 받지 못했습니다`,
		`fetch(activeTask.status_url`,
		`function getReloadTarget(payload, task)`,
		`function scheduleStoryReload(payload, task)`,
		`window.history.replaceState(null, '', target);`,
		`window.location.reload()`,
		`새 내용이 준비되었습니다. 자동으로 최신 화면을 불러옵니다.`,
		`payload.last_completed_job_type`,
		`payload.last_completed_job_turn_id`,
		`Number(payload.last_completed_job_turn_id || 0) > storyTurn`,
		`completedType === 'story_turn'`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in story-room.js asset", want)
		}
	}
}
