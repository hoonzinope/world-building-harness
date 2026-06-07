package harness

const storyRoomTemplateDossier = `
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
      </section>`
