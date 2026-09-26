package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

type User struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	PasswordHash       string     `json:"passwordHash"`
	Role               string     `json:"role"`
	Enabled            bool       `json:"enabled"`
	CreatedAt          time.Time  `json:"createdAt"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
	TotalSeconds       int64      `json:"totalSeconds"`
	MustChangePassword bool       `json:"mustChangePassword,omitempty"`
}
type PublicUser struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	Role               string     `json:"role"`
	Enabled            bool       `json:"enabled"`
	CreatedAt          time.Time  `json:"createdAt"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
	TotalSeconds       int64      `json:"totalSeconds"`
	MustChangePassword bool       `json:"mustChangePassword,omitempty"`
}

func ToPublicUser(u User) PublicUser {
	return PublicUser{ID: u.ID, Username: u.Username, Role: u.Role, Enabled: u.Enabled, CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt, TotalSeconds: u.TotalSeconds, MustChangePassword: u.MustChangePassword}
}

type store struct {
	Users []User `json:"users"`
}
type Service struct {
	mu      sync.Mutex
	path    string
	data    store
	active  *string
	started time.Time
	remote  *remoteClient
}

func New() (*Service, error) {
	if url := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"); url != "" && os.Getenv("SUPABASE_ANON_KEY") != "" {
		return &Service{remote: newRemoteClient(url, os.Getenv("SUPABASE_ANON_KEY"))}, nil
	}
	d, e := os.UserConfigDir()
	if e != nil {
		return nil, e
	}
	s := &Service{path: filepath.Join(d, "AI-Assistant", "auth", "users.json")}
	if e = s.load(); e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, e := os.ReadFile(s.path)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	return json.Unmarshal(b, &s.data)
}
func (s *Service) saveLocked() error {
	if e := os.MkdirAll(filepath.Dir(s.path), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(s.data, "", "  ")
	if e != nil {
		return e
	}
	t := s.path + ".tmp"
	if e = os.WriteFile(t, b, 0600); e != nil {
		return e
	}
	return os.Rename(t, s.path)
}
func id() string { b := make([]byte, 12); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func (s *Service) HasUser(username string) bool {
	if s.remote != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Username == username {
			return true
		}
	}
	return false
}
func (s *Service) HasAdmin() bool {
	if s.remote != nil {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Role == "admin" && u.Enabled {
			return true
		}
	}
	return false
}
func validCredential(username, password string) error {
	if !utf8.ValidString(username) || utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 64 {
		return errors.New("账号长度需为3至64位")
	}
	for _, r := range username {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.') {
			return errors.New("账号包含非法字符")
		}
	}
	if utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return errors.New("密码长度需为8至72字节")
	}
	return nil
}

func (s *Service) BootstrapAdmin(username, password string) error {
	username = strings.TrimSpace(username)
	if s.remote != nil {
		return errors.New("远程模式不支持本地初始化")
	}
	if err := validCredential(username, password); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Role == "admin" {
			return errors.New("管理员已初始化")
		}
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.data.Users = append(s.data.Users, User{ID: id(), Username: username, PasswordHash: string(h), Role: "admin", Enabled: true, CreatedAt: time.Now(), MustChangePassword: false})
	return s.saveLocked()
}

func (s *Service) Create(username, password, role string) error {
	username = strings.TrimSpace(username)
	if s.remote != nil {
		return s.remote.createUser(username, password, "user")
	}
	if err := validCredential(username, password); err != nil {
		return err
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Username == username {
			return errors.New("账号已存在")
		}
	}
	s.data.Users = append(s.data.Users, User{ID: id(), Username: username, PasswordHash: string(h), Role: "user", Enabled: true, CreatedAt: time.Now()})
	return s.saveLocked()
}
func (s *Service) Login(username, password string) (User, error) {
	username = strings.TrimSpace(username)
	if err := validCredential(username, password); err != nil {
		return User{}, errors.New("账号或密码错误")
	}
	if s.remote != nil {
		return s.remote.login(username, password)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Users {
		u := &s.data.Users[i]
		if u.Username == username && u.Enabled && bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil {
			now := time.Now()
			u.LastLoginAt = &now
			s.active = &u.ID
			s.started = now
			_ = s.saveLocked()
			return *u, nil
		}
	}
	return User{}, errors.New("账号或密码错误")
}
func (s *Service) Logout() {
	if s.remote != nil {
		s.remote.logout()
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		for i := range s.data.Users {
			if s.data.Users[i].ID == *s.active {
				s.data.Users[i].TotalSeconds += int64(time.Since(s.started).Seconds())
				break
			}
		}
		s.active = nil
		_ = s.saveLocked()
	}
}
func (s *Service) Current() *User {
	if s.remote != nil {
		return s.remote.current()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		return nil
	}
	for _, u := range s.data.Users {
		if u.ID == *s.active {
			return &u
		}
	}
	return nil
}
func (s *Service) Users() []User {
	if s.remote != nil {
		return s.remote.users()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]User, len(s.data.Users))
	copy(out, s.data.Users)
	return out
}
func (s *Service) ChangePassword(id, oldPassword, newPassword string) error {
	if s.remote != nil {
		return s.remote.changePassword(id, oldPassword, newPassword)
	}
	if utf8.RuneCountInString(newPassword) < 8 || len(newPassword) > 72 {
		return errors.New("密码长度需为8至72字节")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Users {
		if s.data.Users[i].ID != id {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(s.data.Users[i].PasswordHash), []byte(oldPassword)) != nil {
			return errors.New("原密码错误")
		}
		h, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		s.data.Users[i].PasswordHash = string(h)
		s.data.Users[i].MustChangePassword = false
		return s.saveLocked()
	}
	return errors.New("用户不存在")
}
func (s *Service) SetEnabled(id string, enabled bool) error {
	if s.remote != nil {
		return s.remote.setEnabled(id, enabled)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Users {
		if s.data.Users[i].ID != id {
			continue
		}
		if s.data.Users[i].Role == "admin" && !enabled {
			return errors.New("无法停用管理员账号")
		}
		s.data.Users[i].Enabled = enabled
		return s.saveLocked()
	}
	return errors.New("用户不存在")
}

func (s *Service) Delete(id string) error {
	if s.remote != nil {
		return s.remote.deleteUser(id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, u := range s.data.Users {
		if u.ID == id && u.Role != "admin" {
			s.data.Users = append(s.data.Users[:i], s.data.Users[i+1:]...)
			return s.saveLocked()
		}
	}
	return errors.New("无法删除该账号")
}
