package ui

const AdminUsersTemplate = `{{define "content"}}
<h1>Admin Users</h1>
<div class="panel">
  <h2>Create user</h2>
  <form method="post" class="form-grid">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    <input type="hidden" name="action" value="create">
    <div class="field"><label class="field-label" for="admin-create-username">Username</label><input id="admin-create-username" name="username" placeholder="username" required></div>
    <div class="field"><label class="field-label" for="admin-create-display-name">Display name</label><input id="admin-create-display-name" name="display_name" placeholder="display name"></div>
    <div class="field"><label class="field-label" for="admin-create-role">Role</label><select id="admin-create-role" name="role">
      <option>friend</option>
      <option>admin</option>
    </select></div>
    <div class="field"><label class="field-label" for="admin-create-password">Temporary password</label><input id="admin-create-password" name="password" type="password" placeholder="temporary password" required></div>
    <button>create</button>
  </form>
</div>
<table class="table">
  <thead>
    <tr>
      <th>username</th>
      <th>display</th>
      <th>role/status</th>
      <th>last login</th>
      <th>sessions</th>
      <th>actions</th>
    </tr>
  </thead>
  <tbody>
    {{range .Users}}
    <tr>
      <td>
        {{.username}}<br><span class="muted">{{.id}}</span>
      </td>
      <td>{{.display_name}}</td>
      <td>
        <span class="badge">{{.role}}</span> <span class="badge">{{.status}}</span>
      </td>
      <td class="muted">{{.last_login_at}}</td>
      <td>{{.active_sessions}}</td>
      <td>
        <div class="admin-action-grid">
          <form method="post">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="update">
            <input type="hidden" name="id" value="{{.id}}">
            <select name="role" aria-label="{{.username}} role">
              <option {{if eq .role "friend"}}selected{{end}}>friend</option>
              <option {{if eq .role "admin"}}selected{{end}}>admin</option>
            </select>
            <select name="status" aria-label="{{.username}} status">
              <option {{if eq .status "active"}}selected{{end}}>active</option>
              <option {{if eq .status "disabled"}}selected{{end}}>disabled</option>
            </select>
            <button>update</button>
          </form>
          <form method="post">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="reset">
            <input type="hidden" name="id" value="{{.id}}">
            <input name="password" type="password" placeholder="new password" aria-label="{{.username}} new password">
            <button>reset</button>
          </form>
          <form method="post">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="action" value="revoke">
            <input type="hidden" name="id" value="{{.id}}">
            <button class="danger">revoke</button>
          </form>
        </div>
      </td>
    </tr>
    {{end}}
  </tbody>
</table>
{{end}}`
