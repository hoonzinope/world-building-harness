package harness

const loginTemplate = `{{define "content"}}
<section class="auth-shell">
  <div class="auth-panel">
    <div class="auth-panel-head">
      <h1>World Harness</h1>
      <p class="lede">Private story runtime</p>
    </div>
    {{if .Error}}<p class="error" role="alert" id="login-error">{{.Error}}</p>{{end}}
    <form method="post" class="auth-form">
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
      <div class="field">
        <label class="field-label" for="login-username">Username</label>
        <input id="login-username" name="username" autocomplete="username" required autofocus {{if .Error}}aria-invalid="true" aria-describedby="login-error"{{end}}>
      </div>
      <div class="field">
        <label class="field-label" for="login-password">Password</label>
        <input id="login-password" name="password" type="password" autocomplete="current-password" required {{if .Error}}aria-invalid="true" aria-describedby="login-error"{{end}}>
      </div>
      <div class="auth-actions"><button class="primary-button" type="submit">로그인</button></div>
    </form>
  </div>
</section>
{{end}}`
