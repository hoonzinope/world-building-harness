package harness

const foundationStyles = `:root { --paper:#eef2ef; --ink:#1f2523; --muted:#66716d; --line:#cdd7d2; --accent:#b43f34; --deep:#173b37; --wash:#dde7e1; --panel:#ffffff; --ok:#24684b; --warn:#9a6400; --info:#315f99; --shadow:0 16px 42px rgba(17,27,24,.12); }
* { box-sizing:border-box; }
html { scroll-behavior:smooth; }
body { margin:0; background:linear-gradient(180deg,#f4f7f6 0%,#e3e9e5 100%); color:var(--ink); font-family: ui-serif, Georgia, "Apple SD Gothic Neo", "Noto Serif KR", serif; line-height:1.65; }
a { color:var(--deep); text-decoration-thickness:1px; text-underline-offset:3px; }`

const layoutStyles = `.shell { max-width:1180px; margin:0 auto; padding:28px 20px 124px; }
.top { display:flex; align-items:flex-end; justify-content:space-between; gap:20px; border-bottom:1px solid var(--line); padding-bottom:18px; margin-bottom:24px; }
.brand { font-size:13px; letter-spacing:.08em; text-transform:uppercase; color:var(--muted); font-family: ui-sans-serif, system-ui, sans-serif; }
.crumb, .nav { font-family: ui-sans-serif, system-ui, sans-serif; font-size:14px; color:var(--muted); display:flex; gap:12px; flex-wrap:wrap; justify-content:flex-end; }
.nav a { color:var(--deep); }
.nav-form { display:inline-flex; margin:0; }
.link-button { border:0; background:none; color:var(--deep); padding:0; min-height:auto; font:inherit; text-decoration:underline; text-underline-offset:3px; border-radius:0; }
h1 { font-size:clamp(34px, 6vw, 72px); line-height:1; margin:0 0 18px; letter-spacing:0; }
h2 { font-size:24px; margin:34px 0 12px; border-top:1px solid var(--line); padding-top:18px; }
.lede { max-width:760px; font-size:19px; color:#323833; }
.grid { display:grid; grid-template-columns:repeat(auto-fit, minmax(230px, 1fr)); gap:12px; }
.card { border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.36); padding:16px; min-height:118px; }
.card strong { display:block; font-size:19px; margin-bottom:8px; }
.meta { font-family: ui-sans-serif, system-ui, sans-serif; color:var(--muted); font-size:13px; }
.doc-list { columns:2 320px; column-gap:28px; }
.doc-link { break-inside:avoid; display:block; padding:8px 0; border-bottom:1px solid rgba(31,35,33,.08); }
.doc-link span { display:block; color:var(--muted); font-family:ui-sans-serif, system-ui, sans-serif; font-size:12px; }
.reader { display:grid; grid-template-columns:minmax(0, 1fr) 280px; gap:42px; align-items:start; }
.prose { max-width:780px; }
.prose h1 { font-size:44px; margin-top:0; }
.prose h2 { font-size:24px; }
.prose p, .prose li { font-size:18px; }
.side { position:sticky; top:16px; border-left:3px solid var(--accent); padding-left:16px; font-family:ui-sans-serif, system-ui, sans-serif; color:var(--muted); }`

const formStyles = `.search { display:flex; gap:8px; max-width:520px; margin:18px 0 24px; }
.search input, input, select, textarea { border:1px solid var(--line); border-radius:6px; padding:12px 13px; background:var(--panel); font:inherit; width:100%; min-height:44px; max-width:100%; }
textarea { min-height:110px; resize:vertical; }
button, .button { border:1px solid var(--deep); background:var(--deep); color:white; border-radius:6px; padding:11px 15px; min-height:44px; font:600 14px ui-sans-serif, system-ui, sans-serif; cursor:pointer; text-decoration:none; display:inline-flex; align-items:center; justify-content:center; white-space:normal; text-align:center; line-height:1.25; max-width:100%; }
button:focus-visible, .button:focus-visible, input:focus-visible, select:focus-visible, textarea:focus-visible, .link-button:focus-visible, a:focus-visible { outline:3px solid rgba(184,51,45,.45); outline-offset:2px; }
button:disabled { opacity:.45; cursor:not-allowed; }
button.secondary, .button.secondary { background:transparent; color:var(--deep); }
button.danger { border-color:var(--accent); background:var(--accent); }
.filter-bar { display:flex; gap:8px; flex-wrap:wrap; margin:12px 0 18px; }
.filter-link { display:inline-flex; align-items:center; border:1px solid var(--line); border-radius:999px; padding:6px 12px; background:rgba(255,255,255,.38); text-decoration:none; color:var(--deep); font:600 13px ui-sans-serif, system-ui, sans-serif; min-height:36px; }
.filter-link[aria-current="page"], .filter-link[aria-selected="true"] { background:var(--deep); color:#fff; border-color:var(--deep); }
.status-line { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
.story-summary { max-width:44ch; }
.toolbar { display:flex; gap:10px; flex-wrap:wrap; align-items:center; margin:18px 0 22px; }
.table { width:100%; border-collapse:collapse; font-family:ui-sans-serif, system-ui, sans-serif; font-size:14px; }
.table th, .table td { text-align:left; border-bottom:1px solid var(--line); padding:10px 8px; vertical-align:top; }`

const storyLobbyStyles = `.story-lobby-list { display:grid; gap:12px; margin-top:4px; }
.story-card { display:grid; gap:14px; border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.46); padding:16px; box-shadow:0 10px 22px rgba(17,27,24,.04); }
.story-card-head { display:grid; gap:12px; }
.story-card-heading { display:grid; gap:8px; min-width:0; }
.story-card-title { margin:0; font-size:22px; line-height:1.2; word-break:keep-all; overflow-wrap:anywhere; }
.story-card-meta { display:flex; gap:8px; flex-wrap:wrap; align-items:center; font-family:ui-sans-serif, system-ui, sans-serif; color:var(--muted); font-size:13px; }
.story-card-meta .meta { font-size:13px; }
.story-card-badges { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
.story-card-summary { border-left:3px solid var(--accent); background:rgba(255,255,255,.55); border-radius:4px; padding:14px 14px 14px 13px; font-size:16px; line-height:1.75; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.story-card-foot { display:flex; gap:12px; flex-wrap:wrap; align-items:center; justify-content:space-between; }
.story-card-updated { font-size:12px; }
.story-card-actions { display:flex; justify-content:flex-end; flex:0 0 auto; min-width:120px; }
.story-card-actions .button { min-width:120px; }
.badge { display:inline-flex; border:1px solid var(--line); border-radius:999px; padding:2px 8px; font:12px ui-sans-serif, system-ui, sans-serif; color:var(--muted); background:rgba(255,255,255,.35); }`

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
.turn-timeline-link { display:grid; gap:3px; border:1px solid var(--line); border-left:3px solid var(--deep); border-radius:6px; padding:9px 10px; background:rgba(255,255,255,.72); color:var(--ink); text-decoration:none; box-shadow:none; word-break:keep-all; overflow-wrap:anywhere; }
.turn-timeline-link:hover { border-color:rgba(23,59,55,.35); background:rgba(255,255,255,.92); }
.turn-timeline-link[aria-current="true"] { border-color:rgba(49,95,153,.45); background:rgba(49,95,153,.08); border-left-color:var(--info); }
.turn-timeline-turn { font-size:11px; color:var(--muted); }
.turn-timeline-title { font-size:13px; font-weight:600; line-height:1.35; display:-webkit-box; -webkit-line-clamp:2; -webkit-box-orient:vertical; overflow:hidden; }
.previous-turns { display:grid; gap:10px; }
.previous-turn { border:1px solid var(--line); border-radius:6px; background:var(--panel); box-shadow:0 14px 30px rgba(17,27,24,.08); padding:0; scroll-margin-top:18px; }
.previous-turn summary { min-height:60px; cursor:pointer; display:flex; align-items:flex-start; justify-content:space-between; gap:12px; list-style:none; padding:16px 16px 14px; font:700 17px ui-sans-serif, system-ui, sans-serif; }
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
.choice-card { display:grid; grid-template-columns:48px minmax(0, 1fr); text-align:left; justify-content:flex-start; background:var(--panel); color:var(--ink); border-color:var(--line); white-space:normal; align-items:flex-start; gap:12px; width:100%; padding:14px; }
.choice-card:disabled { opacity:.65; }
.choice-card-letter { width:48px; min-width:48px; height:48px; border-radius:6px; display:inline-flex; align-items:center; justify-content:center; background:rgba(23,59,55,.08); color:var(--deep); font:700 16px ui-sans-serif, system-ui, sans-serif; flex:0 0 48px; }
.choice-card-copy { display:grid; gap:4px; min-width:0; }
.choice-card-copy strong { display:block; font-size:15px; line-height:1.55; word-break:keep-all; overflow-wrap:anywhere; }
.choice-card-risk { font:12px ui-sans-serif, system-ui, sans-serif; color:var(--accent); word-break:keep-all; overflow-wrap:anywhere; }
.choice-card-archived { display:grid; grid-template-columns:48px minmax(0, 1fr); gap:12px; border:1px solid var(--line); border-radius:6px; background:var(--panel); padding:14px; }
.choice-card-archived .choice-card-copy { padding-top:2px; }
.choice-card-archived .choice-card-letter { background:rgba(49,95,153,.08); color:var(--info); }
.story-composer { scroll-margin-top:18px; display:grid; gap:12px; }
.story-composer-panel { display:grid; gap:14px; }
.story-choice-submit-panel { display:grid; gap:12px; }
.story-choice-form { margin:0; }
.story-input-form { display:grid; gap:12px; margin:0; }
.story-custom-input-panel { display:grid; gap:12px; }
.story-choice-submit-head,
.story-composer-panel-head { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; justify-content:space-between; }
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
.mode-tabs span { position:relative; z-index:1; display:flex; align-items:center; justify-content:center; min-height:48px; width:100%; padding:8px 12px; border:1px solid var(--line); border-radius:6px; background:var(--panel); font:600 14px ui-sans-serif, system-ui, sans-serif; color:var(--muted); text-align:center; word-break:keep-all; box-sizing:border-box; }
.mode-tabs input:checked + span { border-color:rgba(49,95,153,.45); box-shadow:inset 0 0 0 1px rgba(49,95,153,.08); color:var(--deep); background:rgba(49,95,153,.06); }
.mode-tabs input:focus-visible + span { outline:3px solid rgba(184,51,45,.35); outline-offset:2px; }
.mobile-action-dock { display:none; }
.form-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:12px; }
.panel { border:1px solid var(--line); border-radius:6px; background:var(--panel); padding:16px; margin-bottom:14px; }
.status-panel { border-left:4px solid var(--info); }
.story-progress { display:grid; gap:12px; margin-bottom:0; }
.progress-loader { display:flex; align-items:center; gap:10px; padding-bottom:10px; border-bottom:1px solid rgba(17,27,24,.08); }
.progress-loader-dot { width:12px; height:12px; border-radius:999px; border:2px solid var(--info); background:transparent; flex:0 0 auto; }
.story-progress:not([aria-busy="true"]) .progress-loader-copy { display:none; }
.story-progress[aria-busy="true"] .progress-loader-dot { background:var(--warn); border-color:var(--warn); box-shadow:0 0 0 0 rgba(154,100,0,.26); animation:loaderPulse 1.5s ease-in-out infinite; }
@keyframes loaderPulse { 0% { box-shadow:0 0 0 0 rgba(154,100,0,.26); } 70% { box-shadow:0 0 0 8px rgba(154,100,0,0); } 100% { box-shadow:0 0 0 0 rgba(154,100,0,0); } }
.progress-loader-copy { display:flex; gap:10px; align-items:center; flex-wrap:wrap; min-width:0; }
.progress-loader-copy strong { font:700 16px ui-sans-serif, system-ui, sans-serif; text-transform:lowercase; }
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

const responsiveStyles = `@media (max-width:820px) {
  .shell { padding:16px 14px 176px; }
  .top { align-items:flex-start; flex-direction:column; gap:10px; margin-bottom:18px; }
  .nav { justify-content:flex-start; }
  .reader { grid-template-columns:1fr; }
  .side { position:static; }
  h1 { font-size:38px; }
  .story-room-header { grid-template-columns:1fr; align-items:start; }
  .story-room-grid { grid-template-columns:1fr; grid-template-areas:"current" "timeline" "dossier"; }
  .turn-sidebar,
  .dossier-stack { position:static; }
  .driver-actions { justify-content:flex-start; }
  .current-turn-body { display:flex; flex-direction:column; }
  .current-turn-flow { order:1; }
  .current-turn-body .scene { order:2; }
  .story-choice-submit-head,
  .story-composer-panel-head { align-items:flex-start; }
  .story-choice-submit-head span,
  .story-composer-panel-head span { width:100%; }
  .story-choice-submit-panel .choice-list,
  .story-custom-input-panel,
  .story-composer-actions { width:100%; }
  .story-choice-form .choice-card,
  .story-custom-input-panel textarea,
  .story-composer-actions > * { width:100%; }
  .scene { font-size:17px; line-height:1.72; }
  .panel { padding:14px; }
  .toolbar > *,
  .driver-actions > * { flex:1 1 auto; min-width:0; }
  button,
  .button { width:100%; min-height:48px; }
  .table,
  .table tbody,
  .table tr,
  .table td { display:block; width:100%; }
  .table thead { display:none; }
  .table tr {
    border:1px solid var(--line);
    border-radius:6px;
    background:rgba(255,255,255,.35);
    margin:0 0 12px;
    padding:10px;
  }
  .table td { border:0; padding:6px 4px; }
  .story-lobby-list { gap:10px; }
  .story-card { padding:14px; }
  .story-card-title { font-size:19px; }
  .story-card-summary { font-size:15px; line-height:1.7; }
  .story-card-foot { align-items:stretch; }
  .story-card-actions { width:100%; justify-content:stretch; }
  .story-card-actions .button { width:100%; }
  .mobile-action-dock {
    position:fixed;
    left:0;
    right:0;
    bottom:0;
    z-index:10;
    display:grid;
    grid-template-columns:1fr 1fr;
    gap:8px;
    padding:10px 12px calc(14px + env(safe-area-inset-bottom));
    background:rgba(255,255,255,.94);
    border-top:1px solid var(--line);
    box-shadow:var(--shadow);
    backdrop-filter:blur(12px);
  }
  .mobile-action-dock a { min-height:48px; }
}
@media (max-width:960px) {
  .story-room-grid { grid-template-columns:1fr; grid-template-areas:"current" "timeline" "dossier"; }
  .turn-sidebar,
  .dossier-stack { position:static; }
  .mode-tabs { grid-template-columns:repeat(2, minmax(0, 1fr)); }
  .story-choice-submit-panel .choice-list { grid-template-columns:1fr; }
  .story-choice-form .choice-card { grid-template-columns:48px minmax(0, 1fr); }
  .story-custom-input-panel textarea { min-height:110px; }
  .table { font-size:13px; }
  .turn-timeline-link { max-width:100%; }
}`

const baseStyles = foundationStyles + "\n" + layoutStyles + "\n" + formStyles + "\n" + storyLobbyStyles + "\n" + storyRoomStyles + "\n" + responsiveStyles
