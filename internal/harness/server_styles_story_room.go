package harness

const storyRoomStyles = `.story-room-shell { display:grid; gap:18px; }
.story-room-shell,
.story-room-header > *,
.story-room-grid > *,
.current-turn-column > *,
.turn-sidebar > *,
.dossier-stack,
.dossier-panel,
.story-composer,
.story-composer-panel,
.story-progress,
.turn-timeline a,
.previous-turn,
.choice-card,
.choice-card-archived { min-width:0; }
.story-room-header { display:grid; grid-template-columns:minmax(0, 1fr) auto; gap:14px; align-items:end; padding-bottom:16px; border-bottom:1px solid var(--line); }
.story-room-headline { display:grid; gap:10px; }
.story-room-headline h1 { margin-bottom:0; }
.story-room-meta { display:flex; gap:8px; flex-wrap:wrap; font-family:ui-sans-serif, system-ui, sans-serif; }
.driver-actions { display:flex; gap:8px; flex-wrap:wrap; justify-content:flex-end; }
.story-room-grid { display:grid; grid-template-columns:240px minmax(0, 1fr) 320px; grid-template-areas:"timeline current dossier"; gap:24px; align-items:start; }
.current-turn-column { grid-area:current; display:grid; gap:14px; min-width:0; }
.turn-sidebar { grid-area:timeline; display:grid; gap:12px; min-width:0; position:sticky; top:18px; align-self:start; }
.dossier-stack { grid-area:dossier; display:grid; gap:12px; position:sticky; top:18px; }
.current-turn-panel,
.turn-timeline-panel,
.previous-turns-panel,
.dossier-panel { display:grid; gap:12px; }
.dossier-panel { display:grid; gap:10px; }
.dossier-panel form { display:grid; gap:8px; }
.dossier-panel label { line-height:1.35; }
.admin-action-grid { display:grid; grid-template-columns:repeat(2, minmax(0, 1fr)); gap:8px; }
.admin-action-grid form { width:100%; }
.admin-action-grid button { width:100%; }
.dossier-panel.panel,
.story-composer-panel { margin-bottom:0; }
.turn-timeline { display:grid; gap:8px; font-family:ui-sans-serif, system-ui, sans-serif; }
.turn-timeline-link {
  display:grid;
  gap:3px;
  border:1px solid var(--line);
  border-left:3px solid var(--deep);
  border-radius:6px;
  padding:9px 10px;
  background:rgba(255,255,255,.72);
  color:var(--ink);
  text-decoration:none;
  box-shadow:none;
  word-break:keep-all;
  overflow-wrap:anywhere;
}
.turn-timeline-link:hover { border-color:rgba(23,59,55,.35); background:rgba(255,255,255,.92); }
.turn-timeline-link[aria-current="true"] { border-color:rgba(49,95,153,.45); background:rgba(49,95,153,.08); border-left-color:var(--info); }
.turn-timeline-turn { font-size:11px; color:var(--muted); }
.turn-timeline-title { font-size:13px; font-weight:600; line-height:1.35; display:-webkit-box; -webkit-line-clamp:2; -webkit-box-orient:vertical; overflow:hidden; }
.previous-turns { display:grid; gap:10px; }
.previous-turn { border:1px solid var(--line); border-radius:6px; background:var(--panel); box-shadow:0 14px 30px rgba(17,27,24,.08); padding:0; scroll-margin-top:18px; }
.previous-turn summary {
  min-height:60px;
  cursor:pointer;
  display:flex;
  align-items:flex-start;
  justify-content:space-between;
  gap:12px;
  list-style:none;
  padding:16px 16px 14px;
  font:700 17px ui-sans-serif, system-ui, sans-serif;
}
.previous-turn summary::-webkit-details-marker { display:none; }
.previous-turn summary::after { content:"열기"; color:var(--muted); font:12px ui-sans-serif, system-ui, sans-serif; border:1px solid var(--line); border-radius:999px; padding:4px 8px; flex:0 0 auto; }
.previous-turn[open] summary::after { content:"접기"; }
.previous-turn-head { display:grid; gap:6px; min-width:0; }
.previous-turn-label { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; }
.previous-turn-turn { font-size:14px; color:var(--deep); }
.previous-turn-title { font-size:18px; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.previous-turn-meta { display:flex; gap:8px; flex-wrap:wrap; color:var(--muted); font:12px ui-sans-serif, system-ui, sans-serif; word-break:keep-all; overflow-wrap:anywhere; }
.previous-turn-body { display:grid; gap:14px; padding:0 16px 18px; }
.current-turn-panel { border:1px solid var(--line); border-radius:6px; background:var(--panel); box-shadow:0 14px 30px rgba(17,27,24,.08); padding:18px; scroll-margin-top:18px; }
.current-turn-header { display:grid; gap:8px; padding-bottom:12px; border-bottom:1px solid rgba(17,27,24,.08); }
.current-turn-label { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; }
.current-turn-turn { font-size:14px; color:var(--deep); }
.current-turn-title { font-size:22px; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.current-turn-meta { display:flex; gap:8px; flex-wrap:wrap; color:var(--muted); font:12px ui-sans-serif, system-ui, sans-serif; word-break:keep-all; overflow-wrap:anywhere; }
.current-turn-body { display:grid; gap:16px; padding-top:14px; }
.current-turn-flow { display:grid; gap:14px; border-left:4px solid var(--info); background:rgba(49,95,153,.05); border-radius:6px; padding:14px 14px 14px 16px; }
.turn-section { display:grid; gap:8px; padding-top:12px; border-top:1px solid rgba(17,27,24,.08); }
.turn-section:first-child { padding-top:0; border-top:0; }
.scene { white-space:pre-wrap; font-size:18px; line-height:1.86; max-width:72ch; margin:0; text-wrap:pretty; word-break:keep-all; overflow-wrap:anywhere; }
.choice-list { display:grid; gap:10px; }
.choice-card {
  display:grid;
  grid-template-columns:48px minmax(0, 1fr);
  text-align:left;
  justify-content:flex-start;
  background:var(--panel);
  color:var(--ink);
  border-color:var(--line);
  white-space:normal;
  align-items:flex-start;
  gap:12px;
  width:100%;
  padding:14px;
}
.choice-card:disabled { opacity:.65; }
.choice-card-letter {
  width:48px;
  min-width:48px;
  height:48px;
  border-radius:6px;
  display:inline-flex;
  align-items:center;
  justify-content:center;
  background:rgba(23,59,55,.08);
  color:var(--deep);
  font:700 16px ui-sans-serif, system-ui, sans-serif;
  flex:0 0 48px;
}
.choice-card-copy { display:grid; gap:4px; min-width:0; }
.choice-card-copy strong { display:block; font-size:15px; line-height:1.55; word-break:keep-all; overflow-wrap:anywhere; }
.choice-card-risk { font:12px ui-sans-serif, system-ui, sans-serif; color:var(--accent); word-break:keep-all; overflow-wrap:anywhere; }
.choice-card-archived { display:grid; grid-template-columns:48px minmax(0, 1fr); gap:12px; border:1px solid var(--line); border-radius:6px; background:var(--panel); padding:14px; }
.choice-card-archived .choice-card-copy { padding-top:2px; }
.choice-card-archived .choice-card-letter { background:rgba(49,95,153,.08); color:var(--info); }
.story-composer { scroll-margin-top:18px; display:grid; gap:12px; }
.story-composer-panel { display:grid; gap:14px; border-left:4px solid var(--deep); }
.story-choice-submit-panel { display:grid; gap:12px; }
.story-choice-form { margin:0; }
.story-input-form { display:grid; gap:12px; margin:0; }
.story-custom-input-panel { display:grid; gap:12px; }
.story-choice-submit-head,
.story-composer-panel-head { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; justify-content:space-between; padding-bottom:6px; border-bottom:1px solid rgba(17,27,24,.08); }
.story-choice-submit-head strong,
.story-composer-panel-head strong { font:700 15px ui-sans-serif, system-ui, sans-serif; color:var(--ink); }
.story-input-divider { height:1px; background:rgba(17,27,24,.08); }
.story-composer-actions { margin-top:0; }
.story-choice-submit-panel .choice-list { gap:8px; }
.story-choice-form .choice-card { background:rgba(255,255,255,.72); border-color:var(--line); }
.story-custom-input-panel textarea { min-height:120px; }
.mode-tabs { display:grid; grid-template-columns:repeat(4, minmax(0, 1fr)); gap:8px; }
.mode-tabs label { min-height:48px; display:block; cursor:pointer; position:relative; }
.mode-tabs label:focus-within { outline:3px solid rgba(184,51,45,.35); outline-offset:2px; }
.mode-tabs input { position:absolute; inset:0; opacity:0; margin:0; }
.mode-tabs span {
  position:relative;
  z-index:1;
  display:flex;
  align-items:center;
  justify-content:center;
  min-height:48px;
  width:100%;
  padding:8px 12px;
  border:1px solid var(--line);
  border-radius:6px;
  background:var(--panel);
  font:600 14px ui-sans-serif, system-ui, sans-serif;
  color:var(--muted);
  text-align:center;
  word-break:keep-all;
  box-sizing:border-box;
}
.mode-tabs input:checked + span { border-color:rgba(49,95,153,.45); box-shadow:inset 0 0 0 1px rgba(49,95,153,.08); color:var(--deep); background:rgba(49,95,153,.06); }
.mode-tabs input:focus-visible + span { outline:3px solid rgba(184,51,45,.35); outline-offset:2px; }
.mobile-action-dock { display:none; }
.form-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:12px; }
.panel { border:1px solid var(--line); border-radius:6px; background:var(--panel); padding:16px; margin-bottom:14px; }
.status-panel { border-left:4px solid var(--info); }
.story-progress { display:grid; gap:12px; margin-bottom:0; }
.progress-loader { display:flex; align-items:center; gap:10px; padding-bottom:10px; border-bottom:1px solid rgba(17,27,24,.08); }
.progress-loader-dot { width:12px; height:12px; border-radius:999px; border:2px solid var(--info); background:transparent; flex:0 0 auto; }
.story-progress[aria-busy="true"] .progress-loader-dot { background:var(--warn); border-color:var(--warn); box-shadow:0 0 0 0 rgba(154,100,0,.26); animation:loaderPulse 1.5s ease-in-out infinite; }
@keyframes loaderPulse { 0% { box-shadow:0 0 0 0 rgba(154,100,0,.26); } 70% { box-shadow:0 0 0 8px rgba(154,100,0,0); } 100% { box-shadow:0 0 0 0 rgba(154,100,0,0); } }
.progress-loader-copy { display:flex; gap:10px; align-items:center; flex-wrap:wrap; min-width:0; }
.progress-loader-copy strong { font:700 16px ui-sans-serif, system-ui, sans-serif; text-transform:lowercase; }
.story-progress-steps {
  display:grid;
  grid-template-columns:repeat(4, minmax(0, 1fr));
  gap:8px;
}
.story-progress-step {
  display:grid;
  gap:4px;
  align-content:start;
  border:1px solid var(--line);
  border-radius:6px;
  padding:10px 10px 9px;
  background:rgba(255,255,255,.55);
  color:var(--muted);
  min-height:66px;
}
.story-progress-step-index {
  display:inline-flex;
  align-items:center;
  justify-content:center;
  width:26px;
  height:26px;
  border-radius:999px;
  background:rgba(23,59,55,.08);
  color:var(--deep);
  font:700 12px ui-sans-serif, system-ui, sans-serif;
}
.story-progress-step-text {
  font:600 13px ui-sans-serif, system-ui, sans-serif;
  line-height:1.35;
  word-break:keep-all;
  overflow-wrap:anywhere;
}
.story-progress-step[aria-current="step"],
.story-progress-step.is-active {
  border-color:rgba(49,95,153,.45);
  background:rgba(49,95,153,.08);
  color:var(--deep);
}
.story-progress-step[aria-current="step"] .story-progress-step-index,
.story-progress-step.is-active .story-progress-step-index {
  background:rgba(49,95,153,.12);
  color:var(--info);
}
.story-progress-step.is-complete {
  border-color:rgba(23,59,55,.2);
  background:rgba(23,59,55,.05);
}
.story-progress-step.is-complete .story-progress-step-index {
  background:rgba(23,59,55,.08);
  color:var(--ok);
}
.story-progress[data-step-label="ready"] .progress-loader-copy strong { color:var(--ok); }
.story-progress[data-step-label="queued"] .progress-loader-copy strong { color:var(--info); }
.story-progress[data-step-label="generating"] .progress-loader-copy strong { color:var(--warn); }
.story-progress[data-step-label="applying"] .progress-loader-copy strong { color:var(--deep); }
.story-progress[data-step-label="failed"] .progress-loader-copy strong { color:var(--accent); }
.story-progress-message { margin:0; font:15px ui-sans-serif, system-ui, sans-serif; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.story-progress-meta { margin:0; word-break:keep-all; overflow-wrap:anywhere; }
.story-progress-actions { margin-top:0; }
.story-progress [data-story-refresh] { width:auto; }
[hidden] { display:none !important; }
.input-panel textarea:disabled,
.story-composer-panel textarea:disabled,
.story-composer-panel input:disabled,
.story-composer-panel button:disabled,
.story-progress button:disabled,
.story-progress input:disabled { opacity:.58; cursor:not-allowed; }
.input-panel textarea:disabled { background:rgba(255,255,255,.6); color:var(--muted); }
.story-room-shell [aria-busy="true"] .story-progress { border-left-color:var(--warn); }
.panel h2, .panel h3 { margin-top:0; border:0; padding-top:0; font-family:ui-sans-serif, system-ui, sans-serif; }
.panel ul { padding-left:20px; }
.muted { color:var(--muted); font-family:ui-sans-serif, system-ui, sans-serif; font-size:13px; }
.error { color:var(--accent); font-family:ui-sans-serif, system-ui, sans-serif; }
.empty-state { padding:18px; border:1px dashed var(--line); border-radius:6px; background:rgba(255,255,255,.26); }
.failed-job-meta { display:grid; gap:6px; }`
