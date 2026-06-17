package ui

const storyRoomTemplateDossier = `
    <aside class="dossier-stack" id="story-notes" aria-label="플레이어 노트">
      <section class="dossier-panel panel">
        <div class="dossier-head">
          <div>
            <h2>플레이어 노트</h2>
            <p class="muted">현재 플레이어에게 공개된 정보만 요약합니다.</p>
          </div>
          <span class="badge">제한 공개</span>
        </div>
        {{with .PlayerDossier}}
        <div class="dossier-section">
          <h3>위치</h3>
          {{if .Location}}<p><strong>{{.Location}}</strong></p>{{else}}<p class="muted">아직 확인된 위치가 없습니다.</p>{{end}}
        </div>
        <div class="dossier-section">
          <h3>등장 인물</h3>
          {{if .ActiveCharacters}}
          <div class="dossier-badges">{{range .ActiveCharacters}}<span class="badge">{{.}}</span>{{end}}</div>
          {{else}}
          <p class="muted">현재 장면에 고정된 인물이 없습니다.</p>
          {{end}}
        </div>
        <div class="dossier-section">
          <h3>누적 확인 정보 {{if .HiddenFactCount}}<span class="dossier-hidden-count">숨김 {{.HiddenFactCount}}</span>{{end}}</h3>
          {{if .Facts}}<ul class="dossier-list">{{range .Facts}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="muted">공개된 확인 정보가 없습니다.</p>{{end}}
        </div>
        <div class="dossier-section">
          <h3>열린 실마리 {{if .HiddenThreadCount}}<span class="dossier-hidden-count">숨김 {{.HiddenThreadCount}}</span>{{end}}</h3>
          {{if .OpenThreads}}<ul class="dossier-list">{{range .OpenThreads}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="muted">추적 중인 실마리가 없습니다.</p>{{end}}
        </div>
        <div class="dossier-section">
          <h3>위험 {{if .HiddenRiskCount}}<span class="dossier-hidden-count">숨김 {{.HiddenRiskCount}}</span>{{end}}</h3>
          {{if .Risks}}<ul class="dossier-list">{{range .Risks}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="muted">표시할 위험 신호가 없습니다.</p>{{end}}
        </div>
        {{else}}
        <div class="dossier-section">
          <h3>위치</h3>
          <p><strong>{{.State.Location}}</strong></p>
        </div>
        <div class="dossier-section">
          <h3>등장 인물</h3>
          <div class="dossier-badges">{{range .State.ActiveCharacters}}<span class="badge">{{.}}</span>{{end}}</div>
        </div>
        <div class="dossier-section">
          <h3>누적 확인 정보</h3>
          <ul class="dossier-list">{{range .State.Facts}}<li>{{.}}</li>{{end}}</ul>
        </div>
        <div class="dossier-section">
          <h3>열린 실마리</h3>
          <ul class="dossier-list">{{range .State.OpenThreads}}<li>{{.}}</li>{{end}}</ul>
        </div>
        <div class="dossier-section">
          <h3>위험</h3>
          <ul class="dossier-list">{{range .State.Risks}}<li>{{.}}</li>{{end}}</ul>
        </div>
        {{end}}
      </section>`
