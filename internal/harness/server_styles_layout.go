package harness

const layoutStyles = `
.shell {
  max-width:1180px;
  margin:0 auto;
  padding:28px 20px 124px;
}

.top {
  display:flex;
  align-items:flex-end;
  justify-content:space-between;
  gap:20px;
  border-bottom:1px solid var(--line);
  padding-bottom:18px;
  margin-bottom:24px;
}

.brand {
  font-size:13px;
  letter-spacing:.08em;
  text-transform:uppercase;
  color:var(--muted);
  font-family: ui-sans-serif, system-ui, sans-serif;
}

.crumb,
.nav {
  font-family: ui-sans-serif, system-ui, sans-serif;
  font-size:14px;
  color:var(--muted);
  display:flex;
  gap:12px;
  flex-wrap:wrap;
  justify-content:flex-end;
}

.nav a {
  color:var(--deep);
}

.nav-form {
  display:inline-flex;
  margin:0;
}

.link-button {
  border:0;
  background:none;
  color:var(--deep);
  padding:0;
  min-height:auto;
  font:inherit;
  text-decoration:underline;
  text-underline-offset:3px;
  border-radius:0;
}

h1 {
  font-size:clamp(34px, 6vw, 72px);
  line-height:1;
  margin:0 0 18px;
  letter-spacing:0;
}

h2 {
  font-size:24px;
  margin:34px 0 12px;
  border-top:1px solid var(--line);
  padding-top:18px;
}

.lede {
  max-width:760px;
  font-size:19px;
  color:#323833;
}

.auth-shell {
  display:grid;
  gap:16px;
  justify-items:start;
  max-width:560px;
}

.auth-panel {
  width:100%;
  max-width:420px;
  border:1px solid var(--line);
  border-radius:6px;
  background:var(--panel);
  padding:20px;
  box-shadow:0 14px 30px rgba(17,27,24,.08);
}

.auth-panel-head {
  display:grid;
  gap:8px;
  margin-bottom:18px;
}

.auth-form {
  display:grid;
  gap:14px;
}

.field {
  display:grid;
  gap:8px;
}

.field-label {
  font:600 13px ui-sans-serif, system-ui, sans-serif;
  color:var(--ink);
}

.field-hint {
  font:13px ui-sans-serif, system-ui, sans-serif;
  color:var(--muted);
}

.auth-actions {
  display:flex;
  justify-content:flex-start;
  margin-top:2px;
}

.primary-button {
  min-width:120px;
}

.grid {
  display:grid;
  grid-template-columns:repeat(auto-fit, minmax(230px, 1fr));
  gap:12px;
}

.card {
  border:1px solid var(--line);
  border-radius:6px;
  background:rgba(255,255,255,.36);
  padding:16px;
  min-height:118px;
}

.card strong {
  display:block;
  font-size:19px;
  margin-bottom:8px;
}

.meta {
  font-family: ui-sans-serif, system-ui, sans-serif;
  color:var(--muted);
  font-size:13px;
}

.doc-list {
  columns:2 320px;
  column-gap:28px;
}

.doc-link {
  break-inside:avoid;
  display:block;
  padding:8px 0;
  border-bottom:1px solid rgba(31,35,33,.08);
}

.doc-link span {
  display:block;
  color:var(--muted);
  font-family:ui-sans-serif, system-ui, sans-serif;
  font-size:12px;
}

.reader {
  display:grid;
  grid-template-columns:minmax(0, 1fr) 280px;
  gap:42px;
  align-items:start;
}

.prose {
  max-width:780px;
}

.prose h1 {
  font-size:44px;
  margin-top:0;
}

.prose h2 {
  font-size:24px;
}

.prose p,
.prose li {
  font-size:18px;
}

.side {
  position:sticky;
  top:16px;
  border-left:3px solid var(--accent);
  padding-left:16px;
  font-family:ui-sans-serif, system-ui, sans-serif;
  color:var(--muted);
}`
