package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bobacgo/cron-job/internal/admin"
	"github.com/golang-jwt/jwt/v5"
)

type MgrHandler struct {
	admin *admin.Service
	auth  *AuthHandler
}

func NewMgrHandler(adminSvc *admin.Service, auth *AuthHandler) *MgrHandler {
	return &MgrHandler{admin: adminSvc, auth: auth}
}

func (h *MgrHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}

	user, err := h.admin.GetUserByAccount(r.Context(), strings.TrimSpace(req.Account))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAPIResponse(w, http.StatusOK, 1, nil, "用户名或密码错误")
			return
		}
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	if user.Password != req.Password || user.Status != 1 {
		writeAPIResponse(w, http.StatusOK, 1, nil, "用户名或密码错误")
		return
	}

	if err := h.admin.UpdateUserLogin(r.Context(), user.ID, admin.IPFromRequest(r)); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}

	token, err := h.auth.issueTokenWithRoles(user.Account, admin.RoleIDsFromUser(user))
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, "生成登录令牌失败")
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{"token": token}, "")
}

func (h *MgrHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	user, _, err := h.currentUser(r)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 401, nil, "未授权，请重新登录")
		return
	}

	roles := []string{"all"}
	if user.Account != "admin" {
		roles = []string{"user"}
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{
		"name":  user.Account,
		"roles": roles,
	}, "")
}

func (h *MgrHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) UserList(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	page, pageSize := pageArgs(r)
	resp, err := h.admin.ListUsers(r.Context(), r.URL.Query().Get("keyword"), page, pageSize)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, resp, "")
}

func (h *MgrHandler) UserCreate(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Status   int    `json:"status"`
		RoleIDs  string `json:"role_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	item := &admin.User{
		Account:    req.Account,
		Password:   req.Password,
		Email:      req.Email,
		Phone:      req.Phone,
		Status:     req.Status,
		RoleIDs:    req.RoleIDs,
		RegisterIP: admin.IPFromRequest(r),
		Operator:   current.Account,
	}
	if item.Status == 0 {
		item.Status = 1
	}
	if err := h.admin.CreateUser(r.Context(), item); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	item.Password = ""
	writeAPIResponse(w, http.StatusOK, 0, item, "")
}

func (h *MgrHandler) UserUpdate(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	if err := h.admin.UpdateUserBase(r.Context(), req.ID, req.Phone, req.Email, current.Account); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) UserUpdateStatus(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		ID     int64 `json:"id"`
		Status int   `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	if err := h.admin.UpdateUserStatus(r.Context(), req.ID, req.Status, current.Account); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) UserUpdateRole(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		ID      int64  `json:"id"`
		RoleIDs string `json:"role_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	if err := h.admin.UpdateUserRole(r.Context(), req.ID, req.RoleIDs, current.Account); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) UserUpdatePassword(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		ID          int64  `json:"id"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	if err := h.admin.UpdateUserPassword(r.Context(), req.ID, req.NewPassword, current.Account); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) UserDelete(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	if err := h.admin.DeleteUsers(r.Context(), parseIDs(r.URL.Query().Get("ids"))); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) MenuList(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	page, pageSize := pageArgs(r)
	resp, err := h.admin.ListMenus(r.Context(), r.URL.Query().Get("name"), page, pageSize)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, resp, "")
}

func (h *MgrHandler) MenuTree(w http.ResponseWriter, r *http.Request) {
	_, roleIDs, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	tree, err := h.admin.BuildMenuTree(r.Context(), roleIDs)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{"list": tree}, "")
}

func (h *MgrHandler) MenuCreate(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req admin.Menu
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	req.Operator = current.Account
	if req.MenuType == 0 {
		req.MenuType = 1
	}
	if err := h.admin.CreateMenu(r.Context(), &req); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, req, "")
}

func (h *MgrHandler) MenuUpdate(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req admin.Menu
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	req.Operator = current.Account
	if err := h.admin.UpdateMenu(r.Context(), &req); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, req, "")
}

func (h *MgrHandler) MenuDelete(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	if err := h.admin.DeleteMenus(r.Context(), parseIDs(r.URL.Query().Get("ids"))); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) RoleList(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	page, pageSize := pageArgs(r)
	resp, err := h.admin.ListRoles(r.Context(), r.URL.Query().Get("role_name"), page, pageSize)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, resp, "")
}

func (h *MgrHandler) RoleOne(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	item, err := h.admin.GetRole(r.Context(), id)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, item, "")
}

func (h *MgrHandler) RoleCreate(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req admin.Role
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	req.Operator = current.Account
	if req.Status == 0 {
		req.Status = 1
	}
	if err := h.admin.CreateRole(r.Context(), &req); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, req, "")
}

func (h *MgrHandler) RoleUpdate(w http.ResponseWriter, r *http.Request) {
	current, _, ok := h.mustCurrentUser(w, r)
	if !ok {
		return
	}
	var req admin.Role
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	req.Operator = current.Account
	if err := h.admin.UpdateRole(r.Context(), &req); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, req, "")
}

func (h *MgrHandler) RoleDelete(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	if err := h.admin.DeleteRoles(r.Context(), parseIDs(r.URL.Query().Get("ids"))); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) RolePermissionsGet(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	roleID, _ := strconv.ParseInt(r.URL.Query().Get("role_id"), 10, 64)
	menuIDs, err := h.admin.GetRolePermissions(r.Context(), roleID)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{"menu_ids": menuIDs}, "")
}

func (h *MgrHandler) RolePermissionsPost(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.mustCurrentUser(w, r); !ok {
		return
	}
	var req struct {
		RoleID  int64   `json:"role_id"`
		MenuIDs []int64 `json:"menu_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}
	if err := h.admin.SaveRolePermissions(r.Context(), req.RoleID, req.MenuIDs); err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, err.Error())
		return
	}
	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *MgrHandler) mustCurrentUser(w http.ResponseWriter, r *http.Request) (*admin.User, []int64, bool) {
	user, roleIDs, err := h.currentUser(r)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 401, nil, "未授权，请重新登录")
		return nil, nil, false
	}
	return user, roleIDs, true
}

func (h *MgrHandler) currentUser(r *http.Request) (*admin.User, []int64, error) {
	tokenString := extractBearerToken(r.Header.Get("Authorization"))
	if tokenString == "" {
		return nil, nil, errors.New("missing token")
	}

	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return h.auth.jwtSecret, nil
	})
	if err != nil || !token.Valid || claims.User == "" {
		return nil, nil, errors.New("invalid token")
	}

	user, err := h.admin.GetUserByAccount(r.Context(), claims.User)
	if err != nil {
		return nil, nil, err
	}
	return user, admin.RoleIDsFromUser(user), nil
}

func pageArgs(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	return page, pageSize
}

func parseIDs(value string) []int64 {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil {
			out = append(out, id)
		}
	}
	return out
}
