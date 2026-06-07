package harness

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
