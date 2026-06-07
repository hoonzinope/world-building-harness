package harness

const layoutTemplate = `<!doctype html>
<html lang="ko">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.PageTitle}}</title>
<style>{{.BaseStyles}}</style>
</head>
<body>
<a class="skip-link" href="#main-content">본문으로 건너뛰기</a>
<main class="shell" id="main-content" tabindex="-1">
  <div class="top">
    <a class="brand" href="{{.Base}}/">World Harness</a>
    <div class="nav">
      {{if .StoryEnabled}}
      <a href="{{.Base}}/stories">스토리</a>
      {{end}}
      <a href="{{.Base}}/packs/lumen-federation/">세계관</a>
      {{with .User}}
        {{if eq .Role "admin"}}
        <a href="{{$.Base}}/admin/users">Admin</a>
        {{end}}
        <form class="nav-form" method="post" action="{{$.Base}}/logout">
          <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
          <button class="link-button" type="submit">Logout</button>
        </form>
      {{else}}
        {{if .AuthEnabled}}
        <a href="{{.Base}}/login">로그인</a>
        {{end}}
        <span>{{$.PageTitle}}</span>
      {{end}}
    </div>
  </div>
  {{template "content" .}}
</main>
</body>
</html>`

const indexTemplate = `{{define "content"}}
<h1>World Packs</h1>
<p class="lede">읽기 전용 위키는 pack 단위로 분리되고, 변경은 Telegram/Codex와 world-tool draft workflow를 통해 들어갑니다.</p>
<div class="grid">
  {{range .Packs}}
    <a class="card" href="{{packURL $.Base .id}}">
      <strong>{{.title}}</strong>
      <span class="meta">{{.id}}</span><br>
      <span class="meta">
        {{index .summary "content_documents"}} canon docs · {{index .summary "active_drafts"}} drafts
      </span>
    </a>
  {{else}}
    <div class="card"><strong>No packs</strong><span class="meta">Create packs/&lt;id&gt;/harness.yaml first.</span></div>
  {{end}}
</div>
{{end}}`
