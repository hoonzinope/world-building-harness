package ui

const storyRoomTemplateStart = `{{define "content"}}
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
    <section class="current-turn-column" id="current-scene">
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
    </section>
`
