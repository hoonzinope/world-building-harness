package ui

const storyRoomTemplateAdminPanels = `
      {{if .IsAdmin}}
      <section class="dossier-panel panel">
        <h3>관리</h3>
        <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <input type="hidden" name="action" value="update">
          <label class="muted">상태</label>
          <select name="status">
            <option value="">변경 없음</option>
            <option value="active">진행 중</option>
            <option value="paused">일시 정지</option>
            <option value="completed">완료</option>
            <option value="archived">보관됨</option>
          </select>
          <label class="muted">진행자 ID</label>
          <input name="active_driver_id" placeholder="{{.DriverLabel}}">
          <div class="toolbar"><button>적용</button></div>
        </form>
        <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
          <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
          <input type="hidden" name="action" value="update">
          <input type="hidden" name="active_driver_id" value="__open__">
          <button class="secondary">진행자 비우기</button>
        </form>
        {{if .CanAdminMutate}}
          {{with .LatestTurn}}
          <form method="post" action="{{$.Base}}/stories/{{$.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="edit_turn">
            <label class="muted">현재 턴 {{$.LatestTurnID}} 편집</label>
            <label class="muted">장면 본문</label>
            <textarea name="scene_body">{{.SceneBody}}</textarea>
            <label class="muted">현재 상황</label>
            <textarea name="current_situation">{{.CurrentSituation}}</textarea>
            <div class="toolbar"><button class="secondary">편집 저장</button></div>
          </form>
          {{end}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="rollback_turn">
            <label class="muted">되돌릴 턴</label>
            <select name="turn_id">
              {{range .Turns}}<option value="{{.TurnID}}" {{if eq .TurnID $.LatestTurnID}}selected{{end}}>턴 {{.TurnID}}</option>{{end}}
            </select>
            <div class="toolbar"><button class="secondary">되돌리기</button></div>
          </form>
        {{else if .IsProcessing}}
        <p class="muted">GM 생성 중에는 편집과 롤백을 막습니다.</p>
        {{end}}
        <div class="toolbar admin-action-grid">
          {{if or (eq .Story.Status "archived") (eq .Story.Status "deleted")}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="restore">
            <button>복구</button>
          </form>
          {{else}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="archive">
            <button>보관</button>
          </form>
          {{end}}
          {{if ne .Story.Status "deleted"}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="delete">
            <button class="secondary">삭제</button>
          </form>
          {{end}}
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="export_bundle">
            <button class="secondary">번들 내보내기</button>
          </form>
          <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <input type="hidden" name="action" value="recover_store">
            <button class="secondary">저장소 복구</button>
          </form>
        </div>
      </section>
      {{end}}`
