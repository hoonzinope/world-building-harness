package harness

const packTemplate = `{{define "content"}}
<h1>{{.Title}}</h1>
<p class="lede">{{index .Summary "content_documents"}} canon documents, {{index .Summary "active_drafts"}} active drafts.</p>
  <form class="search" method="get">
    <label class="sr-only" for="pack-search">문서 검색</label>
    <input id="pack-search" name="q" value="{{.Query}}" placeholder="문서 검색">
    <button type="submit">검색</button>
  </form>
{{range .Types}}
  <h2>{{.}}</h2>
  <div class="doc-list">
  {{range index $.Groups .}}
    <a class="doc-link" href="{{docURL $.Base $.Pack .path}}">
      {{.title}}
      <span>{{.path}}</span>
    </a>
  {{end}}
  </div>
{{end}}
{{end}}`

const docTemplate = `{{define "content"}}
<div class="reader">
  <article class="prose">{{.BodyHTML}}</article>
  <aside class="side">
    <div><strong>{{index .Doc "title"}}</strong></div>
    <div>{{index .Doc "type"}} · {{index .Doc "status"}}</div>
    <div>{{index .Doc "path"}}</div>
    <p><a href="{{packURL .Base .Pack}}">세계관 목록으로 돌아가기</a></p>
  </aside>
</div>
{{end}}`
