package harness

const formStyles = `
.search {
  display:flex;
  gap:8px;
  max-width:520px;
  margin:18px 0 24px;
}

.search input,
input,
select,
textarea {
  border:1px solid var(--line);
  border-radius:6px;
  padding:12px 13px;
  background:var(--panel);
  font:inherit;
  width:100%;
  min-height:44px;
  max-width:100%;
}

textarea {
  min-height:110px;
  resize:vertical;
}

button,
.button {
  border:1px solid var(--deep);
  background:var(--deep);
  color:white;
  border-radius:6px;
  padding:11px 15px;
  min-height:44px;
  font:600 14px ui-sans-serif, system-ui, sans-serif;
  cursor:pointer;
  text-decoration:none;
  display:inline-flex;
  align-items:center;
  justify-content:center;
  white-space:normal;
  text-align:center;
  line-height:1.25;
  max-width:100%;
}

button:focus-visible,
.button:focus-visible,
input:focus-visible,
select:focus-visible,
textarea:focus-visible,
.link-button:focus-visible,
a:focus-visible {
  outline:3px solid rgba(184,51,45,.45);
  outline-offset:2px;
}

button:disabled {
  opacity:.45;
  cursor:not-allowed;
}

button.secondary,
.button.secondary {
  background:transparent;
  color:var(--deep);
}

button.danger {
  border-color:var(--accent);
  background:var(--accent);
}

.filter-bar {
  display:flex;
  gap:8px;
  flex-wrap:wrap;
  margin:12px 0 18px;
}

.filter-link {
  display:inline-flex;
  align-items:center;
  border:1px solid var(--line);
  border-radius:999px;
  padding:6px 12px;
  background:rgba(255,255,255,.38);
  text-decoration:none;
  color:var(--deep);
  font:600 13px ui-sans-serif, system-ui, sans-serif;
  min-height:36px;
}

.filter-link[aria-current="page"],
.filter-link[aria-selected="true"] {
  background:var(--deep);
  color:#fff;
  border-color:var(--deep);
}

.status-line {
  display:flex;
  flex-wrap:wrap;
  gap:8px;
  align-items:center;
}

.story-summary {
  max-width:44ch;
}

.toolbar {
  display:flex;
  gap:10px;
  flex-wrap:wrap;
  align-items:center;
  margin:18px 0 22px;
}

.table {
  width:100%;
  border-collapse:collapse;
  font-family:ui-sans-serif, system-ui, sans-serif;
  font-size:14px;
}

.table th,
.table td {
  text-align:left;
  border-bottom:1px solid var(--line);
  padding:10px 8px;
  vertical-align:top;
}`
