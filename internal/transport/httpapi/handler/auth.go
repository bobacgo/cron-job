package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	authCookieName = "cron_job_token"
	defaultJWTKey  = "cron-job-dev-secret"
)

type AuthHandler struct {
	adminUser     string
	adminPassword string
	jwtSecret     []byte
}

type authClaims struct {
	User    string  `json:"user"`
	RoleIDs []int64 `json:"role_ids,omitempty"`
	jwt.RegisteredClaims
}

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func NewAuthHandler() *AuthHandler {
	user := strings.TrimSpace(os.Getenv("ADMIN_USER"))
	if user == "" {
		user = "admin"
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}

	secret := os.Getenv("ADMIN_JWT_SECRET")
	if secret == "" {
		secret = defaultJWTKey + ":" + user
	}

	return &AuthHandler{
		adminUser:     user,
		adminPassword: password,
		jwtSecret:     []byte(secret),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusOK, 400, nil, "请求参数错误")
		return
	}

	if strings.TrimSpace(req.Account) != h.adminUser || req.Password != h.adminPassword {
		writeAPIResponse(w, http.StatusOK, 1, nil, "用户名或密码错误")
		return
	}

	token, err := h.issueToken(req.Account)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 500, nil, "生成登录令牌失败")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	writeAPIResponse(w, http.StatusOK, 0, map[string]any{
		"token": token,
	}, "")
}

func (h *AuthHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticate(r)
	if err != nil {
		writeAPIResponse(w, http.StatusOK, 401, nil, "未授权，请重新登录")
		return
	}

	writeAPIResponse(w, http.StatusOK, 0, map[string]any{
		"name":  user,
		"roles": []string{"all"},
	}, "")
}

func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	writeAPIResponse(w, http.StatusOK, 0, map[string]any{}, "")
}

func (h *AuthHandler) issueToken(user string) (string, error) {
	return h.issueTokenWithRoles(user, nil)
}

func (h *AuthHandler) issueTokenWithRoles(user string, roleIDs []int64) (string, error) {
	now := time.Now()
	claims := authClaims{
		User:    user,
		RoleIDs: roleIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

func (h *AuthHandler) authenticate(r *http.Request) (string, error) {
	tokenString := extractBearerToken(r.Header.Get("Authorization"))
	if tokenString == "" {
		if cookie, err := r.Cookie(authCookieName); err == nil {
			tokenString = cookie.Value
		}
	}
	if tokenString == "" {
		return "", errors.New("missing token")
	}

	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	if claims.User == "" {
		return "", errors.New("empty user")
	}
	return claims.User, nil
}

func (h *AuthHandler) Authenticate(r *http.Request) (string, error) {
	return h.authenticate(r)
}

func (h *AuthHandler) Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.authenticate(r); err != nil {
			writeAPIResponse(w, http.StatusOK, 401, nil, "未授权，请重新登录")
			return
		}
		next(w, r)
	}
}

func extractBearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return ""
}
