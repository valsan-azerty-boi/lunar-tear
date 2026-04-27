package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

//go:embed login.html
var loginFS embed.FS

var loginTmpl = template.Must(template.ParseFS(loginFS, "login.html"))

type Handlers struct {
	store *AuthStore
	tok   *TokenService
}

func NewHandlers(store *AuthStore, tok *TokenService) *Handlers {
	return &Handlers{store: store, tok: tok}
}

type loginPageData struct {
	RedirectURI string
	State       string
	Error       string
	Username    string
}

func isOAuthPath(path string) bool {
	// Match /v{N}/dialog/oauth or /v{N}.{M}/dialog/oauth
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) != 3 {
		return false
	}
	return strings.HasPrefix(parts[0], "v") && parts[1] == "dialog" && parts[2] == "oauth"
}

func isMePath(path string) bool {
	p := strings.TrimPrefix(path, "/")
	if p == "me" {
		return true
	}
	parts := strings.Split(p, "/")
	return len(parts) == 2 && strings.HasPrefix(parts[0], "v") && parts[1] == "me"
}

func (h *Handlers) HandleOAuth(w http.ResponseWriter, r *http.Request) {
	if isMePath(r.URL.Path) {
		h.HandleMe(w, r)
		return
	}

	if !isOAuthPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.oauthGet(w, r)
	case http.MethodPost:
		h.oauthPost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) oauthGet(w http.ResponseWriter, r *http.Request) {
	data := loginPageData{
		RedirectURI: r.URL.Query().Get("redirect_uri"),
		State:       r.URL.Query().Get("state"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := loginTmpl.Execute(w, data); err != nil {
		log.Printf("render login page: %v", err)
	}
}

func (h *Handlers) oauthPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	action := r.FormValue("action")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")

	renderErr := func(msg string) {
		data := loginPageData{
			RedirectURI: redirectURI,
			State:       state,
			Error:       msg,
			Username:    username,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := loginTmpl.Execute(w, data); err != nil {
			log.Printf("render login page: %v", err)
		}
	}

	if username == "" || password == "" {
		renderErr("Username and password are required.")
		return
	}

	var user AuthUser
	var err error

	switch action {
	case "register":
		user, err = h.store.CreateUser(username, password)
		if err == ErrUserExists {
			renderErr("Username is already taken.")
			return
		}
		if err != nil {
			log.Printf("create user: %v", err)
			renderErr("Server error. Try again.")
			return
		}
		log.Printf("registered user %q (id=%d)", user.Username, user.ID)

	case "login":
		user, err = h.store.VerifyUser(username, password)
		if err == ErrInvalidCreds {
			renderErr("Invalid username or password.")
			return
		}
		if err != nil {
			log.Printf("verify user: %v", err)
			renderErr("Server error. Try again.")
			return
		}
		log.Printf("authenticated user %q (id=%d)", user.Username, user.ID)

	default:
		renderErr("Invalid action.")
		return
	}

	token, err := h.tok.Generate(user)
	if err != nil {
		log.Printf("generate token: %v", err)
		renderErr("Server error. Try again.")
		return
	}

	payload := fmt.Sprintf(`{"user_id":"%d"}`, user.ID)
	b64 := base64.StdEncoding.EncodeToString([]byte(payload))

	fragment := url.Values{}
	fragment.Set("access_token", token)
	fragment.Set("token_type", "bearer")
	fragment.Set("expires_in", strconv.FormatInt(int64(tokenTTL.Seconds()), 10))
	fragment.Set("signed_request", "0."+b64)
	if state != "" {
		fragment.Set("state", state)
	}

	target := redirectURI + "?" + fragment.Encode()
	log.Printf("redirecting to %s", target)
	http.Redirect(w, r, target, http.StatusFound)
}

func (h *Handlers) HandleCheckUsername(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	w.Header().Set("Content-Type", "application/json")
	if username == "" {
		json.NewEncoder(w).Encode(map[string]bool{"exists": false})
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"exists": h.store.UserExists(username)})
}

func (h *Handlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("access_token")
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = auth[7:]
		}
	}

	if token == "" {
		http.Error(w, `{"error":{"message":"missing access_token","type":"OAuthException","code":190}}`, http.StatusUnauthorized)
		return
	}

	claims, err := h.tok.Validate(token)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"message":"%s","type":"OAuthException","code":190}}`, err), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":   strconv.FormatInt(claims.Sub, 10),
		"name": claims.Name,
	})
}
