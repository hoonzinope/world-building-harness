package ui

const storyRoomTemplateInputAndQuestions = `
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
          <div class="story-progress-steps" aria-label="진행 단계">
            <span class="story-progress-step" data-story-step="queued"{{if eq .Progress.StepLabel "queued"}} aria-current="step"{{end}}>
              <span class="story-progress-step-index">1</span>
              <span class="story-progress-step-text">대기열</span>
            </span>
            <span class="story-progress-step" data-story-step="generating"{{if eq .Progress.StepLabel "generating"}} aria-current="step"{{end}}>
              <span class="story-progress-step-index">2</span>
              <span class="story-progress-step-text">생성 중</span>
            </span>
            <span class="story-progress-step" data-story-step="applying"{{if eq .Progress.StepLabel "applying"}} aria-current="step"{{end}}>
              <span class="story-progress-step-index">3</span>
              <span class="story-progress-step-text">반영 중</span>
            </span>
            <span class="story-progress-step" data-story-step="ready"{{if eq .Progress.StepLabel "ready"}} aria-current="step"{{end}}>
              <span class="story-progress-step-index">4</span>
              <span class="story-progress-step-text">최신 턴 준비</span>
            </span>
          </div>
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
          <div class="story-composer-panel-head">
            <div>
              <strong>입력 덱</strong>
              <span class="muted">선택 제출과 직접 입력을 한 곳에서 처리합니다.</span>
            </div>
            <span class="badge {{if .Progress.IsProcessing}}busy{{else}}ready{{end}}">{{if .Progress.IsProcessing}}처리 중{{else}}입력 가능{{end}}</span>
          </div>
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
      </section>`
