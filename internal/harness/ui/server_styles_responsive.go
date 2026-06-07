package ui

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
  .story-progress-steps { grid-template-columns:1fr 1fr; }
  .story-choice-submit-panel .choice-list,
  .story-custom-input-panel,
  .story-composer-actions { width:100%; }
  .story-choice-form .choice-card,
  .story-custom-input-panel textarea,
  .story-composer-actions > * { width:100%; }
  .story-card-head,
  .story-card-foot { align-items:flex-start; }
  .story-lobby-header {
    grid-template-columns:1fr;
    align-items:start;
  }
  .story-lobby-actions {
    justify-content:flex-start;
  }
  .story-lobby-actions > .button {
    width:100%;
  }
  .story-lobby-filters {
    width:100%;
  }
  .filter-link {
    flex:1 1 calc(50% - 8px);
  }
  .story-card-badges { justify-content:flex-start; }
  .story-card-actions { width:100%; }
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
  .story-card-foot { align-items:stretch; grid-template-columns:1fr; }
  .story-card-updated { order:1; }
  .story-card-actions { width:100%; justify-content:stretch; }
  .story-card-actions .button { width:100%; }
  .story-card-head { grid-template-columns:1fr; }
  .story-card-badges { justify-content:flex-start; }
  .auth-panel { max-width:none; }
  .auth-actions { width:100%; }
  .auth-actions .primary-button { width:100%; }
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
