package harness

const storyRoomTemplate = `{{define "content"}}
<style>{{.BaseStyles}}</style>
<div
  id="story-room"
  class="story-room-shell"
  data-story-room
  data-story-id="{{.Story.ID}}"
  data-status-url="{{.StatusURL}}"
  data-current-turn="{{.Story.CurrentTurn}}"
  data-initial-processing="{{if .IsProcessing}}true{{else}}false{{end}}"
>
  {{if .IsAnonymous}}
  <div class="panel status-panel">
    <strong>읽기 전용</strong>
    <p>로그인하면 진행, 질문, 진행권, 관리 기능을 사용할 수 있습니다.</p>
  </div>
  {{end}}
  <div class="story-room-header">
    <div class="story-room-headline">
      <h1>{{.Story.Title}}</h1>
      <div class="story-room-meta">
        <span class="badge">{{friendlyStoryStatusLabel .Story.Status}}</span>
        <span class="badge">{{friendlyStoryPhaseLabel .Story.Phase}}</span>
        <span class="badge">턴 {{.Story.CurrentTurn}}</span>
        <span class="badge">진행자 {{.DriverLabel}}</span>
        <span class="badge">{{if .CanDrive}}참여 가능{{else if .CanQuestion}}질문 가능{{else}}읽기 전용{{end}}</span>
      </div>
    </div>
    <div class="driver-actions">
      {{if .CanClaim}}
      <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/driver">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <input type="hidden" name="action" value="claim">
        <button>진행권 받기</button>
      </form>
      {{end}}
      {{if .CanRelease}}
      <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/driver">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <input type="hidden" name="action" value="release">
        <button class="secondary">진행권 내려놓기</button>
      </form>
      {{end}}
    </div>
  </div>
  {{if .IsProcessing}}
  <div class="panel status-panel">
    <strong>GM 생성 중</strong>
    <p>요청이 접수되었습니다. 생성이 끝나면 최신 턴이 갱신됩니다.</p>
  </div>
  {{end}}
  {{if .FailedJob}}
    {{if .FailedJob.CanRecover}}
    <div class="panel status-panel">
      <strong>GM 생성 실패</strong>
      <p>현재 작업이 실패 상태입니다. 복구를 진행하거나 취소할 수 있습니다.</p>
      <div class="toolbar">
        <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/recover">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <input type="hidden" name="action" value="resume">
          <button>resume</button>
        </form>
        <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/recover">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <input type="hidden" name="action" value="cancel">
          <button class="secondary">cancel</button>
        </form>
      </div>
    </div>
    {{else}}
    <div class="panel status-panel">
      <strong>GM 생성 실패</strong>
      <p>현재 작업이 실패 상태입니다. 새 진행 입력은 실패 작업 처리 후 가능합니다.</p>
    </div>
    {{end}}
  {{end}}
  {{if .ExportedBundle}}
  <div class="panel status-panel">
    <strong>Export handoff</strong>
    <p>Bundle exported to <code>{{.ExportedBundle}}</code>.</p>
    <p class="muted">
      Draft creation is pending/manual via the admin writer path. An admin can now create the draft with story export-draft through the writer path.
    </p>
    <p class="muted">
      Target draft: <code>{{.ExportDraftTarget}}</code> · status:
      <span class="badge">{{if .ExportedStatus}}{{.ExportedStatus}}{{else}}draft_pending{{end}}</span>
    </p>
  </div>
  {{end}}
  {{if .RecoveryStatus}}
  <div class="panel status-panel">
    <strong>Store recovery</strong>
    <p>Recovery status: <span class="badge">{{.RecoveryStatus}}</span></p>
    {{if .RecoveryMessage}}<p>{{.RecoveryMessage}}</p>{{end}}
    <p class="muted">
      Checked files:
      {{range $i, $v := .RecoveryChecked}}{{if $i}}, {{end}}<code>{{$v}}</code>{{end}}
    </p>
    {{if .RecoveryRepaired}}
    <p class="muted">
      Repaired items:
      {{range $i, $v := .RecoveryRepaired}}{{if $i}}, {{end}}<code>{{$v}}</code>{{end}}
    </p>
    {{else}}
    <p class="muted">No file tails needed repair.</p>
    {{end}}
    {{if .RecoveryLockRemoved}}<p class="muted">Stale lock.json was removed.</p>{{end}}
  </div>
  {{end}}
  <div class="story-room-grid">
    <section class="current-turn-column">
      {{with .LatestTurn}}
      <section class="current-turn-panel" id="turn-{{.TurnID}}">
        <div class="current-turn-header">
          <div class="current-turn-label">
            <span class="current-turn-turn">현재 턴 {{.TurnID}}</span>
            <span class="current-turn-title">{{sceneJournalTitle .TurnID .SceneTitle}}</span>
          </div>
          <div class="current-turn-meta">
            <span>{{storyTurnTimestamp .CreatedAt}}</span>
            <span>·</span>
            <span>{{friendlyStoryEventKindLabel .Source}}</span>
          </div>
        </div>
        <div class="current-turn-body">
          <div class="scene">{{.SceneBody}}</div>
          <div class="current-turn-flow">
            <div class="turn-section">
              <strong>현재 상황</strong>
              <p>{{.CurrentSituation}}</p>
            </div>
            {{if .RevealedFacts}}
            <div class="turn-section">
              <strong>이번 턴에서 확인된 정보</strong>
              <ul>{{range .RevealedFacts}}<li>{{.}}</li>{{end}}</ul>
            </div>
            {{end}}
            {{if .Choices}}
            <div class="turn-section">
              <strong>다음 갈림길</strong>
              {{if $.IsAnonymous}}
              <div class="choice-list">
                {{range .Choices}}
                <div class="choice-card choice-card-archived">
                  <span class="choice-card-letter">{{.ID}}</span>
                  <span class="choice-card-copy">
                    <strong>{{.Text}}</strong>
                    {{if .RiskHint}}<span class="choice-card-risk">위험: {{.RiskHint}}</span>{{end}}
                  </span>
                </div>
                {{end}}
              </div>
              {{else}}
              <p class="muted">아래 입력 패널에서 선택지 제출 또는 직접 입력을 할 수 있습니다.</p>
              {{end}}
            </div>
            {{end}}
          </div>
        </div>
      </section>
      {{else}}
      <div class="panel current-turn-panel" id="turn-0">
        <strong>아직 턴이 없습니다.</strong>
        <p class="muted">최초 턴이 생성되면 여기에서 본문과 선택지가 표시됩니다.</p>
      </div>
      {{end}}
      <section
        class="story-composer"
        id="input-panel"
        aria-busy="{{if .Progress.IsProcessing}}true{{else}}false{{end}}"
        data-story-input-panel
      >
        <div class="panel status-panel story-progress" id="story-progress" role="status" aria-live="polite" aria-atomic="true" aria-busy="{{if .Progress.IsProcessing}}true{{else}}false{{end}}" data-story-progress data-status-url="{{.StatusURL}}" data-step-index="{{.Progress.StepIndex}}" data-step-label="{{.Progress.StepLabel}}" data-active-job-id="{{.Progress.ActiveJobID}}" data-active-job-status="{{.Progress.ActiveJobStatus}}" data-active-job-type="{{.Progress.ActiveJobType}}" data-next-poll-ms="{{.Progress.NextPollMS}}">
          <div class="progress-loader">
            <span class="progress-loader-dot" aria-hidden="true"></span>
            <div class="progress-loader-copy">
              <strong data-story-progress-label>{{friendlyStoryProgressStepLabel .Progress.StepLabel}}</strong>
              <span class="badge" data-story-progress-status>{{.Progress.StatusLabel}}</span>
            </div>
          </div>
          <p class="story-progress-message" data-story-progress-message>{{.Progress.ProgressMessage}}</p>
          <p class="muted story-progress-meta" data-story-progress-meta hidden>
            <code data-story-progress-job-id>{{.Progress.ActiveJobID}}</code>
            {{if .Progress.ActiveJobType}}<span data-story-progress-job-type>{{.Progress.ActiveJobType}}</span>{{end}}
            {{if .Progress.ActiveJobStatus}}<span data-story-progress-job-status>{{.Progress.ActiveJobStatus}}</span>{{end}}
            {{if gt .Progress.ActiveJobTurnID 0}}<span data-story-progress-turn>{{.Progress.ActiveJobTurnID}}</span>{{end}}
            {{if .Progress.PendingQuestions}}<span data-story-progress-pending-count>{{len .Progress.PendingQuestions}}</span>{{end}}
          </p>
          <div class="toolbar story-progress-actions">
            <button type="button" class="secondary" hidden data-story-refresh>새 내용 표시</button>
          </div>
        </div>
        {{if .CanDrive}}
        <div class="panel story-composer-panel">
          <div class="story-choice-submit-panel">
            <div class="story-choice-submit-head">
              <strong>선택 제출</strong>
              <span class="muted">A/B/C/D를 바로 보낼 수 있습니다.</span>
            </div>
            <div class="choice-list">
              {{with .LatestTurn}}
                {{range .Choices}}
                  {{if .}}
                  <form
                    method="post"
                    action="{{$.Base}}/stories/{{$.Story.ID}}/input"
                    class="story-choice-form"
                    data-story-submit
                    data-story-submit-kind="choice"
                  >
                    <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                    <input type="hidden" name="turn_id" value="{{$.LatestTurnID}}">
                    <input type="hidden" name="idempotency_key" value="{{idem}}">
                    <input type="hidden" name="choice_id" value="{{.ID}}">
                    <button class="choice-card" type="submit" data-story-choice-button {{if $.Progress.IsProcessing}}disabled{{end}}>
                      <span class="choice-card-letter">{{.ID}}</span>
                      <span class="choice-card-copy">
                        <strong>{{.Text}}</strong>
                        {{if .RiskHint}}<span class="choice-card-risk">위험: {{.RiskHint}}</span>{{end}}
                      </span>
                    </button>
                  </form>
                  {{end}}
                {{end}}
              {{end}}
            </div>
          </div>
          <div class="story-input-divider" aria-hidden="true"></div>
          <div class="story-custom-input-panel">
            <div class="story-composer-panel-head">
              <strong>직접 입력</strong>
              <span class="muted">모드 선택 후 내용을 제출합니다.</span>
            </div>
            <form
              method="post"
              action="{{.Base}}/stories/{{.Story.ID}}/input"
              class="story-input-form"
              data-story-submit
              data-story-submit-kind="input"
            >
              <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
              <input type="hidden" name="turn_id" value="{{.LatestTurnID}}">
              <input type="hidden" name="idempotency_key" value="{{idem}}">
              <div class="form-grid">
                <div>
                  <label class="muted">모드</label>
                  <div class="mode-tabs" role="radiogroup" aria-label="입력 모드">
                    <label><input type="radio" name="mode" value="action" checked {{if $.Progress.IsProcessing}}disabled{{end}}><span>행동</span></label>
                    <label><input type="radio" name="mode" value="dialogue" {{if $.Progress.IsProcessing}}disabled{{end}}><span>대사</span></label>
                    <label><input type="radio" name="mode" value="question" {{if $.Progress.IsProcessing}}disabled{{end}}><span>질문</span></label>
                    <label><input type="radio" name="mode" value="narration" {{if $.Progress.IsProcessing}}disabled{{end}}><span>서술 보정</span></label>
                  </div>
                </div>
              </div>
              <textarea
                name="custom_text"
                data-story-custom-textarea
                placeholder="플레이어 캐릭터가 시도하는 행동/대사/서술/질문"
                {{if $.Progress.IsProcessing}}disabled{{end}}
              ></textarea>
              <div class="toolbar story-composer-actions">
                <button type="submit" {{if $.Progress.IsProcessing}}disabled{{end}}>선택 또는 직접 입력 제출</button>
              </div>
            </form>
          </div>
        </div>
        {{else}}
          {{if .IsAnonymous}}
          <p class="muted">로그인하면 진행권을 받고 직접 입력할 수 있습니다.</p>
          {{else if .IsProcessing}}
          <p class="muted">GM 생성 중입니다. 완료되면 자동으로 최신 턴이 갱신됩니다.</p>
          {{else if .CanClaim}}
          <p class="muted">현재 진행권이 비어 있습니다. 진행권을 받은 뒤 입력할 수 있습니다.</p>
          {{else}}
          <p class="muted">현재 {{.DriverLabel}}가 진행 중입니다. 진행 입력은 비활성화되어 있습니다.</p>
          {{end}}
        {{end}}
        <h2 id="qa">질문</h2>
        {{if .CanDrive}}
        <p class="muted">질문은 직접 입력에서 question 모드를 선택해 제출할 수 있습니다.</p>
        {{else if .CanQuestion}}
        <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/question" class="panel story-composer-panel" data-story-submit data-story-submit-kind="question">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <input type="hidden" name="turn_id" value="{{.LatestTurnID}}">
          <input type="hidden" name="idempotency_key" value="{{idem}}">
          <textarea name="question" data-story-question-textarea placeholder="현재 상황, 인물, 단서, 설정, 선택지 의미를 묻는 비진행 질문"></textarea>
          <div class="toolbar"><button class="secondary" type="submit">질문 제출</button></div>
        </form>
        {{else if .IsAnonymous}}
        <p class="muted">로그인하면 질문을 보낼 수 있습니다.</p>
        {{else if .IsProcessing}}
        <p class="muted">GM 생성 중에는 질문 제출도 잠시 막습니다.</p>
        {{else}}
        <p class="muted">completed/archived/deleted room에서는 새 질문을 받지 않습니다.</p>
        {{end}}
        {{range .QA}}
        <div class="panel">
          <div class="muted">{{storyTurnTimestamp .CreatedAt}} · 턴 {{.TurnID}}</div>
          <strong>Q. {{.Question}}</strong>
          <p>A. {{.Answer}}</p>
        </div>
        {{end}}
      </section>
    </section>
    <aside class="turn-sidebar">
      <section class="turn-timeline-panel panel">
        <h2>턴 타임라인</h2>
        {{if .Turns}}
        <nav class="turn-timeline" aria-label="turn timeline">
          {{range .Turns}}
          <a class="turn-timeline-link" href="#turn-{{.TurnID}}"{{if eq .TurnID $.LatestTurnID}} aria-current="true"{{end}}>
            <span class="turn-timeline-turn">턴 {{.TurnID}}</span>
            <span class="turn-timeline-title">{{sceneIndexTitle .TurnID .SceneTitle}}</span>
          </a>
          {{end}}
        </nav>
        {{else}}
        <p class="muted">아직 턴이 없습니다.</p>
        {{end}}
      </section>
      <section class="previous-turns-panel panel">
        <h2>이전 턴</h2>
        {{if .PreviousTurns}}
        <div class="previous-turns">
          {{range .PreviousTurns}}
          <details class="previous-turn" id="turn-{{.TurnID}}">
            <summary>
              <span class="previous-turn-head">
                <span class="previous-turn-label">
                  <span class="previous-turn-turn">턴 {{.TurnID}}</span>
                  <span class="previous-turn-title">{{sceneJournalTitle .TurnID .SceneTitle}}</span>
                </span>
                <span class="previous-turn-meta">
                  <span>{{storyTurnTimestamp .CreatedAt}}</span>
                  <span>·</span>
                  <span>{{friendlyStoryEventKindLabel .Source}}</span>
                </span>
              </span>
            </summary>
            <div class="previous-turn-body">
              <div class="scene">{{.SceneBody}}</div>
              <div class="turn-section">
                <strong>현재 상황</strong>
                <p>{{.CurrentSituation}}</p>
              </div>
              {{if .RevealedFacts}}
              <div class="turn-section">
                <strong>이번 턴에서 확인된 정보</strong>
                <ul>{{range .RevealedFacts}}<li>{{.}}</li>{{end}}</ul>
              </div>
              {{end}}
              {{if .Choices}}
              <div class="turn-section">
                <strong>기록된 선택지</strong>
                <div class="choice-list">
                  {{range .Choices}}
                  <div class="choice-card choice-card-archived">
                    <span class="choice-card-letter">{{.ID}}</span>
                    <span class="choice-card-copy">
                      <strong>{{.Text}}</strong>
                      {{if .RiskHint}}<span class="choice-card-risk">위험: {{.RiskHint}}</span>{{end}}
                    </span>
                  </div>
                  {{end}}
                </div>
              </div>
              {{end}}
            </div>
          </details>
          {{end}}
        </div>
        {{else}}
        <p class="muted">아직 이전 턴이 없습니다.</p>
        {{end}}
      </section>
    </aside>
    <aside class="dossier-stack" aria-label="dossier">
      <section class="dossier-panel panel">
        <h3>위치</h3>
        <p><strong>{{.State.Location}}</strong></p>
      </section>
      <section class="dossier-panel panel">
        <h3>등장 인물</h3>
        <div class="toolbar">{{range .State.ActiveCharacters}}<span class="badge">{{.}}</span>{{end}}</div>
      </section>
      <section class="dossier-panel panel">
        <h3>누적 확인 정보</h3>
        <ul>{{range .State.Facts}}<li>{{.}}</li>{{end}}</ul>
      </section>
      <section class="dossier-panel panel">
        <h3>열린 실마리</h3>
        <ul>{{range .State.OpenThreads}}<li>{{.}}</li>{{end}}</ul>
      </section>
      <section class="dossier-panel panel">
        <h3>위험</h3>
        <ul>{{range .State.Risks}}<li>{{.}}</li>{{end}}</ul>
      </section>
      {{if .IsAdmin}}
      <section class="dossier-panel panel">
        <h3>관리</h3>
        <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <input type="hidden" name="action" value="update">
          <label class="muted">상태</label>
          <select name="status">
            <option value="">변경 없음</option>
            <option value="active">진행 중</option>
            <option value="paused">일시 정지</option>
            <option value="completed">완료</option>
            <option value="archived">보관됨</option>
          </select>
          <label class="muted">진행자 ID</label>
          <input name="active_driver_id" placeholder="{{.DriverLabel}}">
          <div class="toolbar"><button>적용</button></div>
        </form>
        <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <input type="hidden" name="action" value="update">
          <input type="hidden" name="active_driver_id" value="__open__">
          <button class="secondary">진행자 비우기</button>
        </form>
        {{if .CanAdminMutate}}
          {{with .LatestTurn}}
          <form method="post" action="{{$.Base}}/stories/{{$.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="edit_turn">
            <label class="muted">현재 턴 {{$.LatestTurnID}} 편집</label>
            <label class="muted">장면 본문</label>
            <textarea name="scene_body">{{.SceneBody}}</textarea>
            <label class="muted">현재 상황</label>
            <textarea name="current_situation">{{.CurrentSituation}}</textarea>
            <div class="toolbar"><button class="secondary">편집 저장</button></div>
          </form>
          {{end}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="rollback_turn">
            <label class="muted">되돌릴 턴</label>
            <select name="turn_id">
              {{range .Turns}}<option value="{{.TurnID}}" {{if eq .TurnID $.LatestTurnID}}selected{{end}}>턴 {{.TurnID}}</option>{{end}}
            </select>
            <div class="toolbar"><button class="secondary">되돌리기</button></div>
          </form>
        {{else if .IsProcessing}}
        <p class="muted">GM 생성 중에는 편집과 롤백을 막습니다.</p>
        {{end}}
        <div class="toolbar admin-action-grid">
          {{if or (eq .Story.Status "archived") (eq .Story.Status "deleted")}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="restore">
            <button>복구</button>
          </form>
          {{else}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="archive">
            <button>보관</button>
          </form>
          {{end}}
          {{if ne .Story.Status "deleted"}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="delete">
            <button class="secondary">삭제</button>
          </form>
          {{end}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="export_bundle">
            <button class="secondary">번들 내보내기</button>
          </form>
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="recover_store">
            <button class="secondary">저장소 복구</button>
          </form>
        </div>
      </section>
      {{end}}
    </aside>
  </div>
  {{if .HasTurns}}
  <div class="mobile-action-dock">
    <a class="button secondary" href="#turn-{{.LatestTurnID}}">현재 턴</a>
    <a class="button" href="#input-panel">입력/질문</a>
  </div>
  {{else}}
  <div class="mobile-action-dock">
    <a class="button" href="#input-panel">입력/질문</a>
  </div>
  {{end}}
  <script defer src="{{.Base}}/assets/story-room.js"></script>
</div>
{{end}}`

// storyRoomAssetJS is the only story-room client implementation.
// It is served from /assets/story-room.js and loaded by the template via a same-origin script tag.
const storyRoomAssetJS = `(() => {
  const root = document.querySelector('[data-story-room]');
  if (!root) return;
  const progress = root.querySelector('[data-story-progress]');
  if (!progress) return;
  const refreshButton = progress.querySelector('[data-story-refresh]');
  const statusLabel = progress.querySelector('[data-story-progress-label]');
  const statusBadge = progress.querySelector('[data-story-progress-status]');
  const messageNode = progress.querySelector('[data-story-progress-message]');
  const metaNode = progress.querySelector('[data-story-progress-meta]');
  const jobIdNode = progress.querySelector('[data-story-progress-job-id]');
  const jobTypeNode = progress.querySelector('[data-story-progress-job-type]');
  const jobStatusNode = progress.querySelector('[data-story-progress-job-status]');
  const turnNode = progress.querySelector('[data-story-progress-turn]');
  const pendingNode = progress.querySelector('[data-story-progress-pending-count]');
  const stepNodes = Array.from(progress.querySelectorAll('[data-story-step]'));
  const forms = Array.from(root.querySelectorAll('form[data-story-submit]'));
  const inputPanel = root.querySelector('[data-story-input-panel]');
  const storyTurn = Number(root.dataset.currentTurn || 0);
  const initialControlState = new WeakMap();
  let pollTimer = null;
  let activeTask = null;
  let reloadTimer = null;

  function captureInitialControlState(control) {
    if (!control || control.type === 'hidden') return;
    if (!initialControlState.has(control)) {
      initialControlState.set(control, {
        disabled: control.disabled,
        ariaDisabled: control.getAttribute('aria-disabled'),
      });
    }
  }

  function restoreInitialControlState(control) {
    if (!control || control.type === 'hidden') return;
    const initial = initialControlState.get(control);
    if (!initial) return;
    control.disabled = initial.disabled;
    if (initial.disabled) {
      control.setAttribute('aria-disabled', initial.ariaDisabled ?? 'true');
    } else if (initial.ariaDisabled === null) {
      control.removeAttribute('aria-disabled');
    } else {
      control.setAttribute('aria-disabled', initial.ariaDisabled);
    }
    if (control.tagName === 'BUTTON' && control.dataset.storyOriginalHtml) {
      control.innerHTML = control.dataset.storyOriginalHtml;
      delete control.dataset.storyOriginalHtml;
    }
  }

  forms.forEach((form) => {
    form.querySelectorAll('button, input, select, textarea').forEach(captureInitialControlState);
  });

  function setStep(stepLabel) {
    stepNodes.forEach((node) => {
      const active = node.dataset.storyStep === stepLabel;
      node.toggleAttribute('aria-current', active);
      node.classList.toggle('is-active', active);
    });
  }

  function friendlyStepLabel(stepLabel) {
    switch (stepLabel) {
      case 'queued':
        return '대기열';
      case 'generating':
        return '생성 중';
      case 'applying':
        return '반영 중';
      case 'ready':
        return '입력 가능';
      case 'failed':
        return '실패';
      default:
        return stepLabel || '';
    }
  }

  function setBusy(busy) {
    root.setAttribute('aria-busy', busy ? 'true' : 'false');
    progress.setAttribute('aria-busy', busy ? 'true' : 'false');
    if (inputPanel) {
      inputPanel.setAttribute('aria-busy', busy ? 'true' : 'false');
    }
    forms.forEach((form) => {
      form.querySelectorAll('button, input, select, textarea').forEach((control) => {
        if (control.type === 'hidden') return;
        captureInitialControlState(control);
        if (busy) {
          control.disabled = true;
          control.setAttribute('aria-disabled', 'true');
          if (control.tagName === 'BUTTON' && !control.hasAttribute('data-story-choice-button')) {
            if (!control.dataset.storyOriginalHtml) control.dataset.storyOriginalHtml = control.innerHTML;
            control.innerHTML = '처리 중...';
          }
        } else {
          restoreInitialControlState(control);
        }
      });
    });
  }

  function showRefresh(visible) {
    if (!refreshButton) return;
    refreshButton.hidden = !visible;
  }

  function showMeta(visible) {
    if (!metaNode) return;
    metaNode.hidden = !visible;
  }

  function getReloadTarget(payload, task) {
    const payloadTurn = Number(payload && payload.current_turn ? payload.current_turn : 0);
    const activeTurn = Number(payload && payload.active_job_turn_id ? payload.active_job_turn_id : 0);
    const completedTurn = Number(payload && payload.last_completed_job_turn_id ? payload.last_completed_job_turn_id : 0);
    const submittedTurn = Number(task && task.turn_id ? task.turn_id : 0);
    const nextTurn = Math.max(storyTurn, payloadTurn, activeTurn, completedTurn, submittedTurn);
    const completedType = (payload && payload.last_completed_job_type) || (task && task.job_type) || '';
    if (completedType === 'question_answer' || nextTurn <= storyTurn) {
      return '#qa';
    }
    return '#turn-' + nextTurn;
  }

  function scheduleStoryReload(payload, task) {
    if (reloadTimer) {
      window.clearTimeout(reloadTimer);
      reloadTimer = null;
    }
    const target = getReloadTarget(payload, task);
    if (window.location.hash !== target) {
      window.history.replaceState(null, '', target);
    }
    reloadTimer = window.setTimeout(() => window.location.reload(), 420);
  }

  async function readErrorMessage(response) {
    const contentType = (response.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      const payload = await response.json().catch(() => null);
      if (payload && payload.error) {
        return payload.error;
      }
      return 'HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다';
    }
    const text = await response.text().catch(() => '');
    const snippet = text.trim().replace(/\s+/g, ' ').slice(0, 160);
    return 'HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다' + (snippet ? ': ' + snippet : '');
  }

  function renderStatus(payload) {
    const hasMeta = Boolean(
      payload.active_job_id ||
      payload.active_job_type ||
      payload.active_job_status ||
      payload.active_job_turn_id ||
      (payload.pending_questions && payload.pending_questions.length),
    );
    progress.dataset.stepIndex = String(payload.step_index ?? 3);
    progress.dataset.stepLabel = payload.step_label || 'ready';
    progress.dataset.activeJobId = payload.active_job_id || '';
    progress.dataset.activeJobStatus = payload.active_job_status || '';
    progress.dataset.activeJobType = payload.active_job_type || '';
    progress.dataset.nextPollMs = String(payload.next_poll_ms || 0);
    if (statusLabel) statusLabel.textContent = friendlyStepLabel(payload.step_label || (payload.is_processing ? 'generating' : 'ready'));
    if (statusBadge) statusBadge.textContent = payload.status_label || '';
    if (messageNode) messageNode.textContent = payload.progress_message || '';
    if (jobIdNode) jobIdNode.textContent = payload.active_job_id || '';
    if (jobTypeNode) jobTypeNode.textContent = payload.active_job_type || '';
    if (jobStatusNode) jobStatusNode.textContent = payload.active_job_status || '';
    if (turnNode) turnNode.textContent = payload.active_job_turn_id ? String(payload.active_job_turn_id) : '';
    if (pendingNode) pendingNode.textContent = payload.pending_questions ? String(payload.pending_questions.length) : '';
    showMeta(false);
    setStep(payload.step_label || (payload.is_processing ? 'generating' : 'ready'));
    setBusy(Boolean(payload.is_processing));
  }

  async function pollStatus() {
    if (!activeTask || !activeTask.status_url) return;
    try {
      const response = await fetch(activeTask.status_url, {
        headers: {
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        credentials: 'include',
      });
      if (!response.ok) {
        throw new Error(await readErrorMessage(response));
      }
      const payload = await response.json();
      renderStatus(payload);
      const nextPoll = Number(payload.next_poll_ms || activeTask.next_poll_ms || 2500);
      if (payload.is_processing) {
        pollTimer = window.setTimeout(pollStatus, Math.max(1000, nextPoll));
        return;
      }
      showRefresh(true);
      const completedType = payload.last_completed_job_type || activeTask.job_type || payload.active_job_type || '';
      const completedStoryTurn = completedType === 'story_turn' && (
        Number(payload.current_turn || 0) > storyTurn ||
        Number(payload.last_completed_job_turn_id || 0) > storyTurn
      );
      const completedQuestion = completedType === 'question_answer';
      activeTask = null;
      if (payload.active_job_status !== 'failed' && (completedStoryTurn || completedQuestion || Number(payload.current_turn || 0) > storyTurn)) {
        if (messageNode) messageNode.textContent = payload.progress_message || '새 내용이 준비되었습니다. 자동으로 최신 화면을 불러옵니다.';
        scheduleStoryReload(payload, {
          job_type: completedType,
          turn_id: completedStoryTurn ? Number(payload.last_completed_job_turn_id || payload.current_turn || 0) : Number(payload.last_completed_job_turn_id || 0),
        });
      }
    } catch (error) {
      if (messageNode) messageNode.textContent = '상태를 다시 불러오지 못했습니다. 잠시 후 다시 시도해 주세요.';
      pollTimer = window.setTimeout(pollStatus, 2500);
    }
  }

  async function submitForm(form, submitter) {
    if (pollTimer) {
      window.clearTimeout(pollTimer);
      pollTimer = null;
    }
    const data = new FormData(form);
    const requestPayload = Object.fromEntries(data.entries());
    const actionURL = new URL(form.action, window.location.href);
    const requestURL = actionURL.origin === window.location.origin ? actionURL.pathname + actionURL.search : form.action;
    setBusy(true);
    progress.dataset.stepIndex = '0';
    progress.dataset.stepLabel = 'queued';
    if (statusLabel) statusLabel.textContent = friendlyStepLabel('queued');
    setStep('queued');
    if (messageNode) messageNode.textContent = '제출을 보냈습니다. 서버 응답을 기다립니다.';
    showRefresh(false);
    try {
      const response = await fetch(requestURL, {
        method: (form.method || 'post').toUpperCase(),
        body: JSON.stringify(requestPayload),
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
          'X-CSRF-Token': form.querySelector('input[name="csrf_token"]')?.value || '',
        },
      });
      const contentType = (response.headers.get('content-type') || '').toLowerCase();
      const rawBody = await response.text();
      let responsePayload = null;
      if (contentType.includes('application/json')) {
        try {
          responsePayload = JSON.parse(rawBody);
        } catch (parseError) {
          responsePayload = null;
        }
      }
      if (!response.ok || !responsePayload) {
        if (responsePayload && responsePayload.error) {
          throw new Error(responsePayload.error);
        }
        const snippet = rawBody.trim().replace(/\s+/g, ' ').slice(0, 160);
        throw new Error('HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다' + (snippet ? ': ' + snippet : ''));
      }
      activeTask = {
        status_url: responsePayload.status_url,
        next_poll_ms: responsePayload.next_poll_ms || 2500,
        turn_id: responsePayload.turn_id || 0,
        job_id: responsePayload.job_id || '',
        job_type: responsePayload.job_type || '',
      };
      renderStatus(responsePayload);
      pollTimer = window.setTimeout(pollStatus, Math.max(1000, activeTask.next_poll_ms || 2500));
    } catch (error) {
      setBusy(false);
      showRefresh(false);
      if (messageNode) messageNode.textContent = error.message || '제출 처리에 실패했습니다.';
    }
  }

  root.addEventListener('submit', (event) => {
    const form = event.target.closest('form[data-story-submit]');
    if (!form) return;
    event.preventDefault();
    submitForm(form, event.submitter || null);
  });

  if (refreshButton) {
    refreshButton.addEventListener('click', () => window.location.reload());
  }

  if (statusLabel) statusLabel.textContent = friendlyStepLabel(progress.dataset.stepLabel || (root.dataset.initialProcessing === 'true' ? 'generating' : 'ready'));
  setStep(progress.dataset.stepLabel || (root.dataset.initialProcessing === 'true' ? 'generating' : 'ready'));
  setBusy(root.dataset.initialProcessing === 'true');
  if (root.dataset.initialProcessing === 'true') {
    activeTask = {
      status_url: progress.dataset.statusUrl || root.dataset.statusUrl,
      next_poll_ms: Number(progress.dataset.nextPollMs || 2500),
    };
    pollTimer = window.setTimeout(pollStatus, Math.max(1000, activeTask.next_poll_ms || 2500));
  }
})();`
