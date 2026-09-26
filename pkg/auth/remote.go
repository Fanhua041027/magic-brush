package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

const (
	usernameDomain         = "@magicbrush.local"
	maxRemoteResponseBytes = 1 << 20
)

var remoteIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type remoteClient struct {
	mu                  sync.RWMutex
	url, anonKey, token string
	user                *User
	http                *http.Client
}

func newRemoteClient(url, anonKey string) *remoteClient {
	return &remoteClient{url: url, anonKey: anonKey, http: &http.Client{Timeout: 20 * time.Second}}
}
func remoteEmail(username string) string { return username + usernameDomain }
func (c *remoteClient) request(method, path string, body any, auth bool, out any) error {
	return c.requestContext(context.Background(), method, path, body, auth, out)
}
func (c *remoteClient) requestContext(ctx context.Context, method, path string, body any, auth bool, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.url+path, r)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("apikey", c.anonKey)
	if auth {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxRemoteResponseBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxRemoteResponseBytes {
		return errors.New("远程服务响应过大")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("远程服务请求失败 (%d)", response.StatusCode)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return errors.New("远程服务返回无效数据")
		}
	}
	return nil
}

type authResponse struct {
	AccessToken string `json:"access_token"`
	User        struct {
		ID string `json:"id"`
	} `json:"user"`
}
type profileResponse struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	Role               string     `json:"role"`
	Enabled            bool       `json:"enabled"`
	CreatedAt          time.Time  `json:"created_at"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	TotalSeconds       int64      `json:"total_seconds"`
	MustChangePassword bool       `json:"must_change_password"`
}

func (c *remoteClient) login(username, password string) (User, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loginLocked(username, password)
}
func (c *remoteClient) loginLocked(username, password string) (User, error) {
	var a authResponse
	if err := c.request("POST", "/auth/v1/token?grant_type=password", map[string]string{"email": remoteEmail(username), "password": password}, false, &a); err != nil {
		return User{}, errors.New("账号或密码错误")
	}
	if a.AccessToken == "" || !remoteIDPattern.MatchString(a.User.ID) {
		return User{}, errors.New("远程服务返回无效数据")
	}
	oldToken, oldUser := c.token, c.user
	c.token = a.AccessToken
	succeeded := false
	defer func() {
		if !succeeded {
			c.token, c.user = oldToken, oldUser
		}
	}()
	var p []profileResponse
	if err := c.request("GET", "/rest/v1/profiles?select=*&id=eq."+a.User.ID, nil, true, &p); err != nil || len(p) == 0 {
		if err != nil {
			return User{}, err
		}
		return User{}, errors.New("用户资料不存在")
	}
	u := toUser(p[0])
	if !u.Enabled {
		c.token = ""
		c.user = nil
		return User{}, errors.New("账号已停用")
	}
	c.user = &u
	succeeded = true
	return u, nil
}
func toUser(p profileResponse) User {
	return User{ID: p.ID, Username: p.Username, Role: p.Role, Enabled: p.Enabled, CreatedAt: p.CreatedAt, LastLoginAt: p.LastLoginAt, TotalSeconds: p.TotalSeconds, MustChangePassword: p.MustChangePassword}
}
func (c *remoteClient) current() *User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.user == nil {
		return nil
	}
	u := *c.user
	return &u
}
func (c *remoteClient) logout() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.user = nil
}
func (c *remoteClient) createUser(username, password, role string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out struct {
		Error string `json:"error"`
	}
	err := c.request("POST", "/functions/v1/admin-users", map[string]string{"action": "create", "username": username, "password": password}, true, &out)
	if err != nil {
		return err
	}
	return nil
}
func (c *remoteClient) users() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var p []profileResponse
	if c.user == nil || c.user.Role != "admin" || c.request("GET", "/rest/v1/profiles?select=*&order=created_at.asc", nil, true, &p) != nil {
		return []User{}
	}
	out := make([]User, len(p))
	for i := range p {
		out[i] = toUser(p[i])
	}
	return out
}
func (c *remoteClient) changePassword(id, oldPassword, newPassword string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(newPassword) < 8 {
		return errors.New("密码至少8位")
	}
	if c.user == nil {
		return errors.New("请先登录")
	}
	if _, err := c.loginLocked(c.user.Username, oldPassword); err != nil {
		return errors.New("原密码错误")
	}
	if err := c.request("PUT", "/auth/v1/user", map[string]string{"password": newPassword}, true, nil); err != nil {
		return err
	}
	_, err := c.requestProfile(id, map[string]any{"must_change_password": false})
	if err == nil && c.user != nil {
		c.user.MustChangePassword = false
	}
	return err
}
func (c *remoteClient) requestProfile(id string, body map[string]any) ([]profileResponse, error) {
	if !remoteIDPattern.MatchString(id) {
		return nil, errors.New("用户标识无效")
	}
	return nil, c.request("PATCH", "/rest/v1/profiles?id=eq."+id, body, true, nil)
}
func (c *remoteClient) setEnabled(id string, enabled bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.user == nil || c.user.Role != "admin" {
		return errors.New("需要管理员权限")
	}
	_, err := c.requestProfile(id, map[string]any{"enabled": enabled})
	return err
}
func (c *remoteClient) deleteUser(id string) error {
	if !remoteIDPattern.MatchString(id) {
		return errors.New("用户标识无效")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.user == nil || c.user.Role != "admin" {
		return errors.New("需要管理员权限")
	}
	var out struct{}
	return c.request("POST", "/functions/v1/admin-users", map[string]string{"action": "delete", "user_id": id}, true, &out)
}
