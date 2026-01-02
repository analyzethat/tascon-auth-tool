package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"powerbi-access-tool/config"
	"powerbi-access-tool/repository"
)

type Handler struct {
	userRepo   *repository.UserRepository
	accessRepo *repository.AccessRepository
	groupRepo  *repository.GroupRepository
	templates  *template.Template
	config     *config.Config
}

func NewHandler(
	userRepo *repository.UserRepository,
	accessRepo *repository.AccessRepository,
	groupRepo *repository.GroupRepository,
	cfg *config.Config,
) (*Handler, error) {
	tmpl, err := template.ParseGlob(filepath.Join("templates", "*.html"))
	if err != nil {
		return nil, err
	}

	return &Handler{
		userRepo:   userRepo,
		accessRepo: accessRepo,
		groupRepo:  groupRepo,
		templates:  tmpl,
		config:     cfg,
	}, nil
}

func (h *Handler) IndexPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type SettingsPageData struct {
	Server      string
	Database    string
	Username    string
	HasPassword bool
	Saved       bool
}

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	data := SettingsPageData{
		Server:      h.config.Server,
		Database:    h.config.Database,
		Username:    h.config.Username,
		HasPassword: h.config.Password != "",
		Saved:       r.URL.Query().Get("saved") == "1",
	}
	if err := h.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	h.config.Server = r.FormValue("server")
	h.config.Database = r.FormValue("database")
	h.config.Username = r.FormValue("username")

	// Only update password if a new one is provided
	newPassword := r.FormValue("password")
	if newPassword != "" {
		h.config.Password = newPassword
	}

	if err := h.config.Save(); err != nil {
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

func SetupRoutes(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("GET /login", h.LoginPage)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("GET /logout", h.Logout)

	// Pages
	mux.HandleFunc("GET /", h.IndexPage)
	mux.HandleFunc("GET /settings", h.SettingsPage)
	mux.HandleFunc("POST /settings", h.SaveSettings)

	// User API
	mux.HandleFunc("GET /api/users", h.ListUsers)
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.HandleFunc("PUT /api/users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /api/users/{id}", h.DeleteUser)

	// Access API
	mux.HandleFunc("GET /api/users/{id}/access", h.ListUserAccess)
	mux.HandleFunc("POST /api/users/{id}/access", h.AddUserAccess)
	mux.HandleFunc("DELETE /api/access/{id}", h.RemoveAccess)

	// Search API
	mux.HandleFunc("GET /api/groups/search", h.SearchGroups)

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Wrap with auth middleware
	return AuthMiddleware(mux)
}
