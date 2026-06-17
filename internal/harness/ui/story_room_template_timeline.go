package ui

const storyRoomTemplateTimeline = `
    <aside class="turn-sidebar" id="history-panel" aria-label="기록">
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
        {{if .PreviousTurnPreviews}}
        <div class="previous-turns">
          {{range .PreviousTurnPreviews}}
          <details class="previous-turn previous-turn-preview" id="turn-{{.TurnID}}">
            <summary>
              <span class="previous-turn-head">
                <span class="previous-turn-label">
                  <span class="previous-turn-turn">턴 {{.TurnID}}</span>
                  <span class="previous-turn-title">{{if .Title}}{{.Title}}{{else}}세션 기록{{end}}</span>
                </span>
                <span class="previous-turn-meta">
                  <span>{{.Timestamp}}</span>
                  {{if .SourceLabel}}<span>·</span><span>{{.SourceLabel}}</span>{{end}}
                </span>
              </span>
            </summary>
            <div class="previous-turn-body">
              {{if .CurrentSituation}}
              <div class="turn-section">
                <strong>현재 상황</strong>
                <p class="previous-turn-preview-text">{{.CurrentSituation}}</p>
              </div>
              {{end}}
              {{if .Excerpt}}
              <div class="turn-section">
                <strong>기록 요약</strong>
                <p class="previous-turn-excerpt">{{.Excerpt}}</p>
              </div>
              {{end}}
            </div>
          </details>
          {{end}}
        </div>
        {{else if .PreviousTurns}}
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
              <div class="turn-section">
                <strong>기록 요약</strong>
                {{if .CurrentSituation}}
                <p class="previous-turn-excerpt">{{.CurrentSituation}}</p>
                {{else}}
                <p class="muted">요약 준비 중입니다.</p>
                {{end}}
              </div>
              {{if or .RevealedFacts .Choices}}
              <p class="muted previous-turn-counts">
                확인 정보 {{len .RevealedFacts}}건 · 기록된 선택지 {{len .Choices}}개
              </p>
              {{end}}
            </div>
          </details>
          {{end}}
        </div>
        {{else}}
        <p class="muted">아직 이전 턴이 없습니다.</p>
        {{end}}
      </section>
    </aside>`
