package app

import "ai-assistant/pkg/auth"

func (a *App) AuthHasAdmin() bool { return a.authService != nil && a.authService.HasAdmin() }
func (a *App) AuthBootstrapAdmin(username, password string) string {
	if a.authService == nil {
		return "认证服务不可用"
	}
	return errText(a.authService.BootstrapAdmin(username, password))
}
func (a *App) AuthCreateUser(username, password string) string {
	if a.authService == nil {
		return "认证服务不可用"
	}
	if u := a.authService.Current(); u == nil || u.Role != "admin" {
		return "需要管理员权限"
	}
	return errText(a.authService.Create(username, password, "user"))
}
func (a *App) AuthLogin(username, password string) map[string]interface{} {
	if a.authService == nil {
		return map[string]interface{}{"ok": false, "error": "认证服务不可用"}
	}
	u, e := a.authService.Login(username, password)
	if e != nil {
		return map[string]interface{}{"ok": false, "error": e.Error()}
	}
	return map[string]interface{}{"ok": true, "user": map[string]interface{}{"id": u.ID, "username": u.Username, "role": u.Role, "mustChangePassword": u.MustChangePassword}}
}
func (a *App) AuthLogout() {
	if a.authService != nil {
		a.authService.Logout()
	}
}
func (a *App) AuthCurrent() *auth.PublicUser {
	if a.authService == nil {
		return nil
	}
	u := a.authService.Current()
	if u == nil {
		return nil
	}
	public := auth.ToPublicUser(*u)
	return &public
}
func (a *App) AuthUsers() []auth.PublicUser {
	if a.authService == nil {
		return []auth.PublicUser{}
	}
	if u := a.authService.Current(); u == nil || u.Role != "admin" {
		return []auth.PublicUser{}
	}
	users := a.authService.Users()
	out := make([]auth.PublicUser, len(users))
	for i, u := range users {
		out[i] = auth.ToPublicUser(u)
	}
	return out
}
func (a *App) AuthChangePassword(oldPassword, newPassword string) string {
	if a.authService == nil {
		return "认证服务不可用"
	}
	u := a.authService.Current()
	if u == nil {
		return "请先登录"
	}
	return errText(a.authService.ChangePassword(u.ID, oldPassword, newPassword))
}
func (a *App) AuthSetUserEnabled(id string, enabled bool) string {
	if a.authService == nil {
		return "认证服务不可用"
	}
	if u := a.authService.Current(); u == nil || u.Role != "admin" {
		return "需要管理员权限"
	}
	return errText(a.authService.SetEnabled(id, enabled))
}
func (a *App) AuthDeleteUser(id string) string {
	if a.authService == nil {
		return "认证服务不可用"
	}
	if u := a.authService.Current(); u == nil || u.Role != "admin" {
		return "需要管理员权限"
	}
	return errText(a.authService.Delete(id))
}
func errText(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
