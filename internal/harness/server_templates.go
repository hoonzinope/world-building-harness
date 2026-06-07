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

const loginTemplate = `{{define "content"}}
<section class="auth-shell">
  <div class="auth-panel">
    <div class="auth-panel-head">
      <h1>World Harness</h1>
      <p class="lede">Private story runtime</p>
    </div>
    {{if .Error}}<p class="error" role="alert" id="login-error">{{.Error}}</p>{{end}}
    <form method="post" class="auth-form">
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
      <div class="field">
        <label class="field-label" for="login-username">Username</label>
        <input id="login-username" name="username" autocomplete="username" required autofocus {{if .Error}}aria-invalid="true" aria-describedby="login-error"{{end}}>
      </div>
      <div class="field">
        <label class="field-label" for="login-password">Password</label>
        <input id="login-password" name="password" type="password" autocomplete="current-password" required {{if .Error}}aria-invalid="true" aria-describedby="login-error"{{end}}>
      </div>
      <div class="auth-actions"><button class="primary-button" type="submit">로그인</button></div>
    </form>
  </div>
</section>
{{end}}`

const storyLobbyTemplate = `{{define "content"}}
<section class="story-lobby-shell">
  <header class="story-lobby-header">
    <div class="story-lobby-intro">
      <h1>스토리</h1>
      <p class="lede">세계관 문서를 읽고, 실시간 스토리 룸에서 장면 단위로 진행합니다.</p>
      {{if .IsAnonymous}}<p class="story-lobby-note">로그인하지 않아도 스토리 목록과 세계관은 읽을 수 있습니다. 새 스토리 생성과 진행은 로그인 후 가능합니다.</p>{{end}}
    </div>
    <div class="story-lobby-actions">
      {{if .User}}
      <a class="button story-lobby-primary-action" href="{{.Base}}/stories/new">새 스토리</a>
      {{else if .AuthEnabled}}
      <a class="button story-lobby-primary-action" href="{{.Base}}/login">로그인</a>
      {{end}}
      <a class="button secondary story-lobby-refresh-action" href="{{.Base}}/stories">새로고침</a>
    </div>
  </header>
  <nav class="filter-bar story-lobby-filters" role="tablist" aria-label="스토리 필터">
    <a class="filter-link {{if eq .Filter "all"}}is-selected{{end}}" role="tab" aria-selected="{{if eq .Filter "all"}}true{{else}}false{{end}}" href="{{.Base}}/stories" {{if eq .Filter "all"}}aria-current="page"{{end}}>전체</a>
    <a class="filter-link {{if eq .Filter "active"}}is-selected{{end}}" role="tab" aria-selected="{{if eq .Filter "active"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=active" {{if eq .Filter "active"}}aria-current="page"{{end}}>진행 중</a>
    <a class="filter-link {{if eq .Filter "mine"}}is-selected{{end}}" role="tab" aria-selected="{{if eq .Filter "mine"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=mine" {{if eq .Filter "mine"}}aria-current="page"{{end}}>내 스토리</a>
    <a class="filter-link {{if eq .Filter "watch"}}is-selected{{end}}" role="tab" aria-selected="{{if eq .Filter "watch"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=watch" {{if eq .Filter "watch"}}aria-current="page"{{end}}>관전</a>
    <a class="filter-link {{if eq .Filter "archived"}}is-selected{{end}}" role="tab" aria-selected="{{if eq .Filter "archived"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=archived" {{if eq .Filter "archived"}}aria-current="page"{{end}}>보관됨</a>
    <a class="filter-link {{if eq .Filter "imported"}}is-selected{{end}}" role="tab" aria-selected="{{if eq .Filter "imported"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=imported" {{if eq .Filter "imported"}}aria-current="page"{{end}}>가져온 스토리</a>
  </nav>
  <div class="story-lobby-list" role="list" aria-label="스토리 세션 목록">
  {{range .Stories}}
    <article class="story-card" role="listitem">
      <div class="story-card-head">
        <div class="story-card-heading">
          <h2 class="story-card-title">{{.Title}}</h2>
          <div class="story-card-meta">
            <span class="meta">{{.MetaLine}}</span>
          </div>
        </div>
        <div class="story-card-badges">
          <span class="badge">{{friendlyStoryStatusLabel .Status}}</span>
          <span class="badge">{{storyLobbyPhaseLabel .Phase}}</span>
          <span class="badge">{{.Permission}}</span>
        </div>
      </div>
      <div class="story-card-summary">{{.Summary}}</div>
      <div class="story-card-foot">
        <div class="story-card-updated muted">업데이트 {{.Updated}}</div>
        <div class="story-card-actions">
          <a class="button" href="{{storyURL $.Base .ID}}">입장하기</a>
        </div>
      </div>
    </article>
  {{else}}
    <div class="panel empty-state story-lobby-empty">아직 story room이 없습니다.</div>
  {{end}}
</div>
</section>
{{end}}`

const newStoryTemplate = `{{define "content"}}
<h1>새 스토리</h1>
<form method="post" class="panel">
  <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
  <div class="form-grid">
    <div><label class="muted" for="new-story-world">세계관</label><input id="new-story-world" value="lumen-federation" disabled></div>
    <div><label class="muted" for="new-story-title">제목</label><input id="new-story-title" name="title" placeholder="새 스토리"></div>
    <div>
      <label class="muted" for="new-story-style">스타일</label>
      <select id="new-story-style" name="style">
        <option value="조사극">조사극</option>
        <option value="생존극">생존극</option>
        <option value="행정/법정극">행정/법정극</option>
        <option value="앙상블">앙상블</option>
        <option value="자유">자유</option>
      </select>
    </div>
    <div><label class="muted" for="new-story-character-name">캐릭터 이름</label><input id="new-story-character-name" name="character_name" placeholder="캐릭터 이름"></div>
  </div>
  <label class="muted" for="new-story-traits">특징 / 취향</label>
  <textarea id="new-story-traits" name="traits" placeholder="캐릭터 특징, 보고 싶은 장면 압력, 피하고 싶은 톤"></textarea>
  <div class="toolbar"><button>프롤로그 생성</button><a class="button secondary" href="{{.Base}}/stories">취소</a></div>
</form>
{{end}}`

const adminUsersTemplate = `{{define "content"}}
<h1>Admin Users</h1>
<div class="panel">
  <h2>Create user</h2>
  <form method="post" class="form-grid">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    <input type="hidden" name="action" value="create">
    <div class="field"><label class="field-label" for="admin-create-username">Username</label><input id="admin-create-username" name="username" placeholder="username" required></div>
    <div class="field"><label class="field-label" for="admin-create-display-name">Display name</label><input id="admin-create-display-name" name="display_name" placeholder="display name"></div>
    <div class="field"><label class="field-label" for="admin-create-role">Role</label><select id="admin-create-role" name="role">
      <option>friend</option>
      <option>admin</option>
    </select></div>
    <div class="field"><label class="field-label" for="admin-create-password">Temporary password</label><input id="admin-create-password" name="password" type="password" placeholder="temporary password" required></div>
    <button>create</button>
  </form>
</div>
<table class="table">
  <thead>
    <tr>
      <th>username</th>
      <th>display</th>
      <th>role/status</th>
      <th>last login</th>
      <th>sessions</th>
      <th>actions</th>
    </tr>
  </thead>
  <tbody>
    {{range .Users}}
    <tr>
      <td>
        {{.username}}<br><span class="muted">{{.id}}</span>
      </td>
      <td>{{.display_name}}</td>
      <td>
        <span class="badge">{{.role}}</span> <span class="badge">{{.status}}</span>
      </td>
      <td class="muted">{{.last_login_at}}</td>
      <td>{{.active_sessions}}</td>
      <td>
        <div class="admin-action-grid">
          <form method="post">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="update">
            <input type="hidden" name="id" value="{{.id}}">
            <select name="role" aria-label="{{.username}} role">
              <option {{if eq .role "friend"}}selected{{end}}>friend</option>
              <option {{if eq .role "admin"}}selected{{end}}>admin</option>
            </select>
            <select name="status" aria-label="{{.username}} status">
              <option {{if eq .status "active"}}selected{{end}}>active</option>
              <option {{if eq .status "disabled"}}selected{{end}}>disabled</option>
            </select>
            <button>update</button>
          </form>
          <form method="post">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="reset">
            <input type="hidden" name="id" value="{{.id}}">
            <input name="password" type="password" placeholder="new password" aria-label="{{.username}} new password">
            <button>reset</button>
          </form>
          <form method="post">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="revoke">
            <input type="hidden" name="id" value="{{.id}}">
            <button class="secondary">revoke sessions</button>
          </form>
        </div>
      </td>
    </tr>
    {{end}}
  </tbody>
</table>
{{end}}`
