package harness

const storyRoomTemplateTimeline = `
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
    </aside>`
