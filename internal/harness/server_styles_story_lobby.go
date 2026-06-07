package harness

const storyLobbyStyles = `
.story-lobby-list {
  display:grid;
  gap:12px;
  margin-top:4px;
}

.story-lobby-shell {
  display:grid;
  gap:18px;
}

.story-lobby-header {
  display:grid;
  grid-template-columns:minmax(0, 1fr) auto;
  gap:14px;
  align-items:end;
}

.story-lobby-intro {
  display:grid;
  gap:8px;
}

.story-lobby-note {
  margin:0;
  font:14px ui-sans-serif, system-ui, sans-serif;
  color:var(--muted);
  max-width:760px;
}

.story-lobby-actions {
  display:flex;
  gap:8px;
  flex-wrap:wrap;
  justify-content:flex-end;
}

.story-lobby-actions > .button {
  min-width:120px;
}

.story-lobby-filters {
  display:flex;
  gap:8px;
  flex-wrap:wrap;
  align-items:center;
  margin:0;
}

.filter-link {
  display:inline-flex;
  align-items:center;
  justify-content:center;
  min-height:40px;
  padding:8px 12px;
  border:1px solid var(--line);
  border-radius:6px;
  background:rgba(255,255,255,.55);
  color:var(--muted);
  text-decoration:none;
  font:600 13px ui-sans-serif, system-ui, sans-serif;
}

.filter-link.is-selected,
.filter-link[aria-current="page"] {
  border-color:rgba(49,95,153,.45);
  background:rgba(49,95,153,.08);
  color:var(--deep);
}

.story-card {
  display:grid;
  gap:14px;
  border:1px solid var(--line);
  border-radius:6px;
  background:rgba(255,255,255,.46);
  padding:16px;
  box-shadow:0 10px 22px rgba(17,27,24,.04);
}

.story-card-head {
  display:grid;
  grid-template-columns:minmax(0, 1fr) auto;
  gap:12px;
  align-items:start;
}

.story-card-heading {
  display:grid;
  gap:8px;
  min-width:0;
}

.story-card-title {
  margin:0;
  font-size:22px;
  line-height:1.2;
  word-break:keep-all;
  overflow-wrap:anywhere;
}

.story-card-meta {
  display:flex;
  gap:8px;
  flex-wrap:wrap;
  align-items:center;
  font-family:ui-sans-serif, system-ui, sans-serif;
  color:var(--muted);
  font-size:13px;
}

.story-card-meta .meta {
  font-size:13px;
}

.story-card-badges {
  display:flex;
  gap:8px;
  flex-wrap:wrap;
  align-items:flex-start;
  justify-content:flex-end;
}

.story-card-summary {
  border-left:3px solid var(--accent);
  background:rgba(255,255,255,.55);
  border-radius:4px;
  padding:14px 14px 14px 13px;
  font-size:15px;
  line-height:1.75;
  color:var(--ink);
  word-break:keep-all;
  overflow-wrap:anywhere;
}

.story-card-foot {
  display:grid;
  grid-template-columns:minmax(0, 1fr) auto;
  gap:12px;
  flex-wrap:wrap;
  align-items:end;
  justify-content:space-between;
  padding-top:12px;
  border-top:1px solid rgba(17,27,24,.08);
}

.story-card-updated {
  font-size:12px;
  line-height:1.55;
}

.story-card-actions {
  display:flex;
  justify-content:flex-end;
  flex:0 0 auto;
  min-width:120px;
}

.story-card-actions .button {
  min-width:120px;
}

.badge {
  display:inline-flex;
  border:1px solid var(--line);
  border-radius:999px;
  padding:2px 8px;
  font:12px ui-sans-serif, system-ui, sans-serif;
  color:var(--muted);
  background:rgba(255,255,255,.35);
}`
