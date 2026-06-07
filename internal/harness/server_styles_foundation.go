package harness

const foundationStyles = `
:root {
  --paper:#eef2ef;
  --ink:#1f2523;
  --muted:#66716d;
  --line:#cdd7d2;
  --accent:#b43f34;
  --deep:#173b37;
  --wash:#dde7e1;
  --panel:#ffffff;
  --ok:#24684b;
  --warn:#9a6400;
  --info:#315f99;
  --shadow:0 16px 42px rgba(17,27,24,.12);
}

* {
  box-sizing:border-box;
}

html {
  scroll-behavior:smooth;
}

body {
  margin:0;
  background:linear-gradient(180deg,#f4f7f6 0%,#e3e9e5 100%);
  color:var(--ink);
  font-family: ui-serif, Georgia, "Apple SD Gothic Neo", "Noto Serif KR", serif;
  line-height:1.65;
}

.skip-link {
  position:absolute;
  left:16px;
  top:12px;
  transform:translateY(-180%);
  padding:10px 14px;
  background:var(--panel);
  border:2px solid var(--deep);
  border-radius:6px;
  color:var(--deep);
  font-family: ui-sans-serif, system-ui, sans-serif;
  font-size:14px;
  z-index:20;
}

.skip-link:focus,
.skip-link:focus-visible {
  transform:translateY(0);
}

.sr-only {
  position:absolute;
  width:1px;
  height:1px;
  padding:0;
  margin:-1px;
  overflow:hidden;
  clip:rect(0, 0, 0, 0);
  white-space:nowrap;
  border:0;
}

a {
  color:var(--deep);
  text-decoration-thickness:1px;
  text-underline-offset:3px;
}`
