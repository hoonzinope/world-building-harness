package ui

const storyRoomTemplateEnd = `
  </div>
  {{if .HasTurns}}
  <div class="mobile-action-dock">
    <a class="button secondary" href="#turn-{{.LatestTurnID}}">현재 턴</a>
    <a class="button" href="#input-panel">입력/질문</a>
  </div>
  {{else}}
  <div class="mobile-action-dock">
    <a class="button" href="#input-panel">입력/질문</a>
  </div>
  {{end}}
  <script defer src="{{.Base}}/assets/story-room.js"></script>
</div>
{{end}}`
