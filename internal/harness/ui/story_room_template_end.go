package ui

const storyRoomTemplateEnd = `
  </div>
  {{if .HasTurns}}
  <div class="mobile-action-dock" aria-label="핵심 구역 이동">
    <a class="button secondary" href="#current-scene">장면</a>
    <a class="button" href="#input-panel" aria-label="입력/질문">입력</a>
    <a class="button secondary" href="#qa">질문</a>
    <a class="button secondary" href="#history-panel">기록</a>
  </div>
  {{else}}
  <div class="mobile-action-dock" aria-label="핵심 구역 이동">
    <a class="button secondary" href="#current-scene">장면</a>
    <a class="button" href="#input-panel" aria-label="입력/질문">입력</a>
    <a class="button secondary" href="#qa">질문</a>
    <a class="button secondary" href="#story-notes">노트</a>
  </div>
  {{end}}
  <script defer src="{{.Base}}/assets/story-room.js"></script>
</div>
{{end}}`
