package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bobacgo/cron-job/kit/sqlx"
)

type Service struct {
	db *sqlx.DB
}

type User struct {
	ID         int64  `json:"id"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	Account    string `json:"account"`
	Password   string `json:"password,omitempty"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Status     int    `json:"status"`
	RegisterAt int64  `json:"register_at"`
	RegisterIP string `json:"register_ip"`
	LoginAt    int64  `json:"login_at"`
	LoginIP    string `json:"login_ip"`
	RoleIDs    string `json:"role_ids"`
	Operator   string `json:"operator"`
}

type Role struct {
	ID          int64  `json:"id"`
	RoleName    string `json:"role_name"`
	Description string `json:"description"`
	UserCount   int64  `json:"user_count,omitempty"`
	Status      int    `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	Operator    string `json:"operator"`
}

type Menu struct {
	ID        int64           `json:"id"`
	ParentID  int64           `json:"parent_id"`
	MenuType  int             `json:"menu_type"`
	Path      string          `json:"path"`
	Name      string          `json:"name"`
	Component string          `json:"component"`
	Redirect  string          `json:"redirect"`
	Meta      json.RawMessage `json:"meta"`
	Icon      string          `json:"icon"`
	Sort      int             `json:"sort"`
	RoleIDs   string          `json:"role_ids"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
	Operator  string          `json:"operator"`
	Children  []*Menu         `json:"children,omitempty"`
}

type PageResp[T any] struct {
	Total int64 `json:"total"`
	List  []*T  `json:"list"`
}

func New(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Bootstrap(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT NOT NULL AUTO_INCREMENT,
			account VARCHAR(64) NOT NULL,
			password VARCHAR(255) NOT NULL,
			phone VARCHAR(32) NOT NULL DEFAULT '',
			email VARCHAR(128) NOT NULL DEFAULT '',
			status TINYINT NOT NULL DEFAULT 1,
			register_at BIGINT NOT NULL DEFAULT 0,
			register_ip VARCHAR(64) NOT NULL DEFAULT '',
			login_at BIGINT NOT NULL DEFAULT 0,
			login_ip VARCHAR(64) NOT NULL DEFAULT '',
			role_ids VARCHAR(255) NOT NULL DEFAULT '',
			operator VARCHAR(64) NOT NULL DEFAULT '',
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (id),
			UNIQUE KEY uk_users_account (account)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS roles (
			id BIGINT NOT NULL AUTO_INCREMENT,
			role_name VARCHAR(64) NOT NULL,
			description VARCHAR(255) NOT NULL DEFAULT '',
			status TINYINT NOT NULL DEFAULT 1,
			operator VARCHAR(64) NOT NULL DEFAULT '',
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (id),
			UNIQUE KEY uk_roles_role_name (role_name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS menus (
			id BIGINT NOT NULL AUTO_INCREMENT,
			parent_id BIGINT NOT NULL DEFAULT 0,
			menu_type TINYINT NOT NULL DEFAULT 1,
			path VARCHAR(255) NOT NULL,
			name VARCHAR(128) NOT NULL,
			component VARCHAR(255) NOT NULL DEFAULT '',
			redirect VARCHAR(255) NOT NULL DEFAULT '',
			meta JSON NULL,
			icon VARCHAR(128) NOT NULL DEFAULT '',
			sort INT NOT NULL DEFAULT 0,
			role_ids VARCHAR(255) NOT NULL DEFAULT '',
			operator VARCHAR(64) NOT NULL DEFAULT '',
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return s.seed(ctx)
}

func (s *Service) seed(ctx context.Context) error {
	now := time.Now().Unix()

	var roleCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM roles`).Scan(&roleCount); err != nil {
		return err
	}
	if roleCount == 0 {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO roles (id, role_name, description, status, operator, created_at, updated_at) VALUES (1, 'admin', '系统管理员', 1, 'system', ?, ?)`, now, now); err != nil {
			return err
		}
	}

	var userCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO users (id, account, password, phone, email, status, register_at, register_ip, login_at, login_ip, role_ids, operator, created_at, updated_at) VALUES (1, 'admin', 'admin', '', '', 1, ?, '127.0.0.1', 0, '', '1', 'system', ?, ?)`, now, now, now); err != nil {
			return err
		}
	}

	var menuCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM menus`).Scan(&menuCount); err != nil {
		return err
	}
	if menuCount == 0 {
		menus := []struct {
			id, parentID                                         int64
			path, name, component, redirect, icon, roleIDs, meta string
			sort                                                 int
		}{
			{10, 0, "/mgr", "mgr", "LAYOUT", "/mgr/user", "setting", "1", `{"title":{"zh_CN":"系统管理","en_US":"System"}}`, 10},
			{11, 10, "user", "MgrUser", "/mgr/user/index", "", "user", "1", `{"title":{"zh_CN":"用户管理","en_US":"Users"}}`, 0},
			{12, 10, "role", "MgrRole", "/mgr/role/index", "", "secured", "1", `{"title":{"zh_CN":"角色管理","en_US":"Roles"}}`, 1},
			{13, 10, "menu", "MgrMenu", "/mgr/menu/index", "", "root-list", "1", `{"title":{"zh_CN":"菜单管理","en_US":"Menus"}}`, 2},
		}
		for _, m := range menus {
			if _, err := s.db.ExecContext(ctx, `INSERT INTO menus (id, parent_id, menu_type, path, name, component, redirect, meta, icon, sort, role_ids, operator, created_at, updated_at) VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, 'system', ?, ?)`,
				m.id, m.parentID, m.path, m.name, m.component, m.redirect, m.meta, m.icon, m.sort, m.roleIDs, now, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) GetUserByAccount(ctx context.Context, account string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, created_at, updated_at, account, password, phone, email, status, register_at, register_ip, login_at, login_ip, role_ids, operator FROM users WHERE account = ? LIMIT 1`, account)
	var user User
	if err := row.Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.Account, &user.Password, &user.Phone, &user.Email, &user.Status, &user.RegisterAt, &user.RegisterIP, &user.LoginAt, &user.LoginIP, &user.RoleIDs, &user.Operator); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) UpdateUserLogin(ctx context.Context, id int64, loginIP string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET login_at = ?, login_ip = ?, updated_at = ? WHERE id = ?`, time.Now().Unix(), loginIP, time.Now().Unix(), id)
	return err
}

func (s *Service) ListUsers(ctx context.Context, keyword string, page, pageSize int) (*PageResp[User], error) {
	where := ""
	args := make([]any, 0)
	if keyword != "" {
		where = " WHERE account LIKE ? OR phone LIKE ? OR email LIKE ?"
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	total, err := s.count(ctx, `SELECT COUNT(*) FROM users`+where, args...)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, created_at, updated_at, account, password, phone, email, status, register_at, register_ip, login_at, login_ip, role_ids, operator FROM users` + where + ` ORDER BY id DESC`
	if page > 0 && pageSize > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pageSize, (page-1)*pageSize)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*User, 0)
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt, &item.Account, &item.Password, &item.Phone, &item.Email, &item.Status, &item.RegisterAt, &item.RegisterIP, &item.LoginAt, &item.LoginIP, &item.RoleIDs, &item.Operator); err != nil {
			return nil, err
		}
		item.Password = ""
		list = append(list, &item)
	}
	return &PageResp[User]{Total: total, List: list}, rows.Err()
}

func (s *Service) CreateUser(ctx context.Context, user *User) error {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `INSERT INTO users (account, password, phone, email, status, register_at, register_ip, login_at, login_ip, role_ids, operator, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, 0, '', ?, ?, ?, ?)`,
		user.Account, user.Password, user.Phone, user.Email, user.Status, now, user.RegisterIP, user.RoleIDs, user.Operator, now, now)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	user.ID = id
	return nil
}

func (s *Service) UpdateUserBase(ctx context.Context, id int64, phone, email, operator string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET phone = ?, email = ?, operator = ?, updated_at = ? WHERE id = ?`, phone, email, operator, time.Now().Unix(), id)
	return err
}

func (s *Service) UpdateUserStatus(ctx context.Context, id int64, status int, operator string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET status = ?, operator = ?, updated_at = ? WHERE id = ?`, status, operator, time.Now().Unix(), id)
	return err
}

func (s *Service) UpdateUserRole(ctx context.Context, id int64, roleIDs, operator string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET role_ids = ?, operator = ?, updated_at = ? WHERE id = ?`, roleIDs, operator, time.Now().Unix(), id)
	return err
}

func (s *Service) UpdateUserPassword(ctx context.Context, id int64, password, operator string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET password = ?, operator = ?, updated_at = ? WHERE id = ?`, password, operator, time.Now().Unix(), id)
	return err
}

func (s *Service) DeleteUsers(ctx context.Context, ids []int64) error {
	return s.deleteByIDs(ctx, `DELETE FROM users WHERE id IN (%s)`, ids)
}

func (s *Service) ListRoles(ctx context.Context, roleName string, page, pageSize int) (*PageResp[Role], error) {
	where := ""
	args := make([]any, 0)
	if roleName != "" {
		where = " WHERE role_name LIKE ?"
		args = append(args, roleName+"%")
	}
	total, err := s.count(ctx, `SELECT COUNT(*) FROM roles`+where, args...)
	if err != nil {
		return nil, err
	}
	query := `SELECT r.id, r.role_name, r.description, r.status, r.created_at, r.updated_at, r.operator, COUNT(u.id) FROM roles r LEFT JOIN users u ON FIND_IN_SET(r.id, u.role_ids)` + where + ` GROUP BY r.id, r.role_name, r.description, r.status, r.created_at, r.updated_at, r.operator ORDER BY r.id DESC`
	if page > 0 && pageSize > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pageSize, (page-1)*pageSize)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*Role, 0)
	for rows.Next() {
		var item Role
		if err := rows.Scan(&item.ID, &item.RoleName, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.Operator, &item.UserCount); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	return &PageResp[Role]{Total: total, List: list}, rows.Err()
}

func (s *Service) GetRole(ctx context.Context, id int64) (*Role, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, role_name, description, status, created_at, updated_at, operator FROM roles WHERE id = ? LIMIT 1`, id)
	var item Role
	if err := row.Scan(&item.ID, &item.RoleName, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.Operator); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) CreateRole(ctx context.Context, role *Role) error {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `INSERT INTO roles (role_name, description, status, operator, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		role.RoleName, role.Description, role.Status, role.Operator, now, now)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	role.ID = id
	return nil
}

func (s *Service) UpdateRole(ctx context.Context, role *Role) error {
	_, err := s.db.ExecContext(ctx, `UPDATE roles SET role_name = ?, description = ?, status = ?, operator = ?, updated_at = ? WHERE id = ?`,
		role.RoleName, role.Description, role.Status, role.Operator, time.Now().Unix(), role.ID)
	return err
}

func (s *Service) DeleteRoles(ctx context.Context, ids []int64) error {
	return s.deleteByIDs(ctx, `DELETE FROM roles WHERE id IN (%s)`, ids)
}

func (s *Service) ListMenus(ctx context.Context, name string, page, pageSize int) (*PageResp[Menu], error) {
	where := ""
	args := make([]any, 0)
	if name != "" {
		where = " WHERE name LIKE ? OR path LIKE ?"
		like := "%" + name + "%"
		args = append(args, like, like)
	}
	total, err := s.count(ctx, `SELECT COUNT(*) FROM menus`+where, args...)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, parent_id, menu_type, path, name, component, redirect, meta, icon, sort, role_ids, created_at, updated_at, operator FROM menus` + where + ` ORDER BY sort ASC, id ASC`
	if page > 0 && pageSize > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pageSize, (page-1)*pageSize)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*Menu, 0)
	for rows.Next() {
		item, err := scanMenu(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return &PageResp[Menu]{Total: total, List: list}, rows.Err()
}

func (s *Service) CreateMenu(ctx context.Context, menu *Menu) error {
	now := time.Now().Unix()
	if len(menu.Meta) == 0 {
		menu.Meta = json.RawMessage(`{}`)
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO menus (parent_id, menu_type, path, name, component, redirect, meta, icon, sort, role_ids, operator, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		menu.ParentID, menu.MenuType, menu.Path, menu.Name, menu.Component, menu.Redirect, string(menu.Meta), menu.Icon, menu.Sort, menu.RoleIDs, menu.Operator, now, now)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	menu.ID = id
	return nil
}

func (s *Service) UpdateMenu(ctx context.Context, menu *Menu) error {
	if len(menu.Meta) == 0 {
		menu.Meta = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE menus SET parent_id = ?, path = ?, name = ?, component = ?, redirect = ?, meta = ?, icon = ?, sort = ?, operator = ?, updated_at = ? WHERE id = ?`,
		menu.ParentID, menu.Path, menu.Name, menu.Component, menu.Redirect, string(menu.Meta), menu.Icon, menu.Sort, menu.Operator, time.Now().Unix(), menu.ID)
	return err
}

func (s *Service) DeleteMenus(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	all := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		all[id] = struct{}{}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, parent_id FROM menus`)
	if err != nil {
		return err
	}
	defer rows.Close()
	children := make(map[int64][]int64)
	for rows.Next() {
		var id, parentID int64
		if err := rows.Scan(&id, &parentID); err != nil {
			return err
		}
		children[parentID] = append(children[parentID], id)
	}
	var walk func(int64)
	walk = func(id int64) {
		for _, childID := range children[id] {
			if _, ok := all[childID]; ok {
				continue
			}
			all[childID] = struct{}{}
			walk(childID)
		}
	}
	for _, id := range ids {
		walk(id)
	}
	finalIDs := make([]int64, 0, len(all))
	for id := range all {
		finalIDs = append(finalIDs, id)
	}
	return s.deleteByIDs(ctx, `DELETE FROM menus WHERE id IN (%s)`, finalIDs)
}

func (s *Service) BuildMenuTree(ctx context.Context, roleIDs []int64) ([]*Menu, error) {
	list, err := s.listAllMenus(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]*Menu, 0, len(list))
	for _, item := range list {
		if allowMenu(item.RoleIDs, roleIDs) {
			filtered = append(filtered, item)
		}
	}
	return buildMenuTree(filtered), nil
}

func (s *Service) GetRolePermissions(ctx context.Context, roleID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, role_ids FROM menus ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roleStr := strconv.FormatInt(roleID, 10)
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		var roleIDs string
		if err := rows.Scan(&id, &roleIDs); err != nil {
			return nil, err
		}
		if containsCSV(roleIDs, roleStr) {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func (s *Service) SaveRolePermissions(ctx context.Context, roleID int64, menuIDs []int64) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, parent_id, role_ids FROM menus ORDER BY id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type menuRole struct {
		ID       int64
		ParentID int64
		RoleIDs  string
	}
	items := make([]menuRole, 0)
	for rows.Next() {
		var item menuRole
		if err := rows.Scan(&item.ID, &item.ParentID, &item.RoleIDs); err != nil {
			return err
		}
		items = append(items, item)
	}
	roleStr := strconv.FormatInt(roleID, 10)
	want := make(map[int64]struct{}, len(menuIDs))
	for _, id := range menuIDs {
		want[id] = struct{}{}
	}
	for _, item := range items {
		roleIDs := splitCSV(item.RoleIDs)
		roleIDs = removeCSV(roleIDs, roleStr)
		if _, ok := want[item.ID]; ok {
			roleIDs = appendCSV(roleIDs, roleStr)
		}
		_, err := s.db.ExecContext(ctx, `UPDATE menus SET role_ids = ?, updated_at = ? WHERE id = ?`, strings.Join(roleIDs, ","), time.Now().Unix(), item.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) listAllMenus(ctx context.Context) ([]*Menu, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, parent_id, menu_type, path, name, component, redirect, meta, icon, sort, role_ids, created_at, updated_at, operator FROM menus ORDER BY sort ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*Menu, 0)
	for rows.Next() {
		item, err := scanMenu(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func scanMenu(scanner interface{ Scan(dest ...any) error }) (*Menu, error) {
	var item Menu
	var meta sql.NullString
	if err := scanner.Scan(&item.ID, &item.ParentID, &item.MenuType, &item.Path, &item.Name, &item.Component, &item.Redirect, &meta, &item.Icon, &item.Sort, &item.RoleIDs, &item.CreatedAt, &item.UpdatedAt, &item.Operator); err != nil {
		return nil, err
	}
	if meta.Valid && meta.String != "" {
		item.Meta = json.RawMessage(meta.String)
	} else {
		item.Meta = json.RawMessage(`{}`)
	}
	return &item, nil
}

func buildMenuTree(list []*Menu) []*Menu {
	nodeMap := make(map[int64]*Menu, len(list))
	roots := make([]*Menu, 0)
	for _, item := range list {
		item.Children = make([]*Menu, 0)
		nodeMap[item.ID] = item
	}
	for _, item := range list {
		if item.ParentID == 0 {
			roots = append(roots, item)
			continue
		}
		parent, ok := nodeMap[item.ParentID]
		if !ok {
			roots = append(roots, item)
			continue
		}
		parent.Children = append(parent.Children, item)
	}
	return roots
}

func allowMenu(roleIDs string, userRoleIDs []int64) bool {
	if roleIDs == "" {
		return true
	}
	if len(userRoleIDs) == 0 {
		return false
	}
	allowed := splitCSV(roleIDs)
	for _, roleID := range userRoleIDs {
		if containsCSV(strings.Join(allowed, ","), strconv.FormatInt(roleID, 10)) {
			return true
		}
	}
	return false
}

func (s *Service) count(ctx context.Context, query string, args ...any) (int64, error) {
	var total int64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Service) deleteByIDs(ctx context.Context, pattern string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	holders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		holders = append(holders, "?")
		args = append(args, id)
	}
	query := fmt.Sprintf(pattern, strings.Join(holders, ","))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func containsCSV(v string, target string) bool {
	for _, item := range splitCSV(v) {
		if item == target {
			return true
		}
	}
	return false
}

func removeCSV(items []string, target string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != target {
			out = append(out, item)
		}
	}
	return out
}

func appendCSV(items []string, target string) []string {
	for _, item := range items {
		if item == target {
			return items
		}
	}
	return append(items, target)
}

func RoleIDsFromUser(user *User) []int64 {
	parts := splitCSV(user.RoleIDs)
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err == nil {
			result = append(result, id)
		}
	}
	return result
}

func IPFromRequest(r *http.Request) string {
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		return host[:idx]
	}
	return host
}
