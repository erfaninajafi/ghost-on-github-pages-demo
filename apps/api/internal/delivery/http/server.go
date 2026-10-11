// Package http exposes use cases over versioned JSON endpoints.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	stdhttp "net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/platform"
	"expertsacademy/blog/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type userKey struct{}
type Server struct {
	Service *usecase.Service
	Redis   platform.Redis
	Config  platform.Config
	Log     *slog.Logger
	Ready   func(context.Context) error
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type errorResponse struct {
	Error apiError `json:"error"`
}

func write[T interface{}](w stdhttp.ResponseWriter, status int, v T) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w stdhttp.ResponseWriter, status int, code, message string) {
	write(w, status, errorResponse{apiError{code, message}})
}
func (s *Server) resultError(w stdhttp.ResponseWriter, e error) {
	switch {
	case errors.Is(e, domain.ErrTooLarge):
		fail(w, 413, "image_too_large", "تصویر باید حداکثر ۵ مگابایت و ۱۲ میلیون پیکسل باشد.")
	case errors.Is(e, domain.ErrInvalid):
		fail(w, 400, "invalid_input", "اطلاعات واردشده معتبر نیست.")
	case errors.Is(e, domain.ErrUnauthorized):
		fail(w, 401, "unauthorized", "لطفاً وارد حساب خود شوید.")
	case errors.Is(e, domain.ErrForbidden):
		fail(w, 403, "forbidden", "اجازهٔ انجام این کار را ندارید.")
	case errors.Is(e, domain.ErrNotFound):
		fail(w, 404, "not_found", "موردی یافت نشد.")
	case errors.Is(e, domain.ErrConflict):
		fail(w, 409, "conflict", "اطلاعات تکراری یا ارتباط نامعتبر است.")
	default:
		s.Log.Error("request failed", "error", e)
		fail(w, 500, "internal_error", "خطای داخلی؛ دوباره تلاش کنید.")
	}
}
func decode[T interface{}](w stdhttp.ResponseWriter, r *stdhttp.Request, v *T) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return domain.ErrInvalid
	}
	r.Body = stdhttp.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return domain.ErrInvalid
	}
	var extra json.RawMessage
	if e := d.Decode(&extra); e != io.EOF {
		return domain.ErrInvalid
	}
	return nil
}
func current(r *stdhttp.Request) *domain.User {
	v, _ := r.Context().Value(userKey{}).(*domain.User)
	return v
}
func (s *Server) Handler() stdhttp.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, s.logging, s.security, middleware.Timeout(30*time.Second))
	r.Get("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		write(w, 200, struct {
			Status string `json:"status"`
		}{"ok"})
	})
	r.Get("/readyz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if e := s.Ready(ctx); e != nil {
			fail(w, 503, "not_ready", "dependencies unavailable")
			return
		}
		write(w, 200, struct {
			Status string `json:"status"`
		}{"ready"})
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.rateLimit, s.authenticate)
		r.Post("/auth/signup", s.signup)
		r.Post("/auth/login", s.login)
		r.Post("/auth/logout", s.logout)
		r.Get("/posts", s.posts)
		r.Get("/posts/popular", s.popular)
		r.Get("/posts/search-suggest", s.suggest)
		r.Get("/posts/{slug}", s.detail)
		r.Get("/posts/{id}/comments", s.comments)
		r.Post("/posts/{id}/comments", s.addComment)
		r.Get("/taxonomy", s.taxonomy)
		r.Get("/media/{owner}/{file}", s.media)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuthor)
			r.Get("/auth/me", func(w stdhttp.ResponseWriter, r *stdhttp.Request) { write(w, 200, current(r)) })
			r.Get("/studio/posts", s.ownPosts)
			r.Post("/studio/posts", s.savePost)
			r.Post("/studio/media", s.uploadImage)
			r.Group(func(r chi.Router) {
				r.Use(s.requireAdmin)
				r.Get("/studio/authors", s.authors)
				r.Post("/studio/authors", s.createAuthor)
				r.Patch("/studio/authors/{id}", s.updateAuthor)
				r.Patch("/studio/posts/{id}/status", s.changePostStatus)
				r.Delete("/studio/posts/{id}", s.deletePost)
				r.Get("/studio/comments", s.pending)
				r.Patch("/studio/comments/{id}", s.moderate)
				r.Post("/studio/taxonomy/{kind}", s.addTaxon)
			})
		})
	})
	return r
}
func (s *Server) logging(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.Log.Info("http request", "request_id", middleware.GetReqID(r.Context()), "method", r.Method, "route", chi.RouteContext(r.Context()).RoutePattern(), "status", ww.Status(), "duration_ms", time.Since(start).Milliseconds())
	})
}
func (s *Server) security(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-store")
		origin := r.Header.Get("Origin")
		if origin != "" && origin != s.Config.Origin {
			fail(w, 403, "origin_denied", "Origin is not allowed")
			return
		}
		if origin == s.Config.Origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(204)
			return
		}
		// JSON-only writes and an exact Origin check protect cookie sessions from CSRF.
		if r.Method != "GET" && r.Method != "HEAD" && origin != s.Config.Origin {
			fail(w, 403, "csrf_rejected", "A matching Origin header is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func clientIP(r *stdhttp.Request) string {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		host = r.RemoteAddr
	}
	// Only explicitly trusted reverse proxies may supply forwarded client addresses.
	for _, cidr := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		_, network, e := net.ParseCIDR(strings.TrimSpace(cidr))
		if e == nil && network.Contains(net.ParseIP(host)) {
			parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
			ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1]))
			if ip != nil {
				return ip.String()
			}
		}
	}
	return host
}
func (s *Server) rateLimit(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		max := int64(180)
		bucket := "read"
		if r.Method != "GET" {
			max = 20
			bucket = "write"
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") && r.Method == "POST" {
			max = 8
			bucket = "auth"
		}
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		ok, e := s.Redis.Allow(ctx, clientIP(r)+":"+bucket, max)
		if e != nil {
			if r.Method != "GET" {
				fail(w, 503, "rate_limit_unavailable", "موقتاً امکان ثبت درخواست نیست.")
				return
			}
			s.Log.Warn("read rate limiter unavailable", "error", e)
		} else if !ok {
			w.Header().Set("Retry-After", "60")
			fail(w, 429, "rate_limited", "کمی بعد دوباره تلاش کنید.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) authenticate(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		// Login and logout must remain available when an old session has expired.
		if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") && r.URL.Path != "/api/v1/auth/me" {
			next.ServeHTTP(w, r)
			return
		}
		raw := ""
		bearer := false
		if c, e := r.Cookie("ea_session"); e == nil {
			raw = c.Value
		}
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			raw = strings.TrimPrefix(h, "Bearer ")
			bearer = true
		}
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		// Expired browser cookies should not make public reading require login.
		// Explicit Bearer credentials and all write/studio operations still fail closed.
		rejectSession := func() {
			if !bearer {
				s.clearSession(w)
				if r.Method == "GET" && !strings.HasPrefix(r.URL.Path, "/api/v1/studio/") && r.URL.Path != "/api/v1/auth/me" {
					next.ServeHTTP(w, r)
					return
				}
			}
			s.resultError(w, domain.ErrUnauthorized)
		}
		claims := &jwt.RegisteredClaims{}
		token, e := jwt.ParseWithClaims(raw, claims, func(_ *jwt.Token) (interface{}, error) { return []byte(s.Config.JWTSecret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("expertsacademy-api"), jwt.WithAudience("expertsacademy-web"), jwt.WithExpirationRequired())
		if e != nil || !token.Valid || !usecase.ValidID(claims.Subject) {
			rejectSession()
			return
		}
		u, e := s.Service.Repo.UserByID(r.Context(), claims.Subject)
		if e != nil {
			if errors.Is(e, domain.ErrNotFound) {
				rejectSession()
				return
			}
			s.resultError(w, e)
			return
		}
		if u.DisabledAt != nil {
			rejectSession()
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, &u)))
	})
}
func (s *Server) requireAuthor(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		u := current(r)
		if u == nil {
			s.resultError(w, domain.ErrUnauthorized)
			return
		}
		if u.Role != "author" && u.Role != "admin" {
			s.resultError(w, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) requireAdmin(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		u := current(r)
		if u == nil {
			s.resultError(w, domain.ErrUnauthorized)
			return
		}
		if u.Role != "admin" || u.DisabledAt != nil {
			s.resultError(w, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) session(w stdhttp.ResponseWriter, u domain.User) {
	now := time.Now()
	claims := jwt.RegisteredClaims{Subject: u.ID, Issuer: "expertsacademy-api", Audience: jwt.ClaimStrings{"expertsacademy-web"}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(8 * time.Hour)), ID: uuid.NewString()}
	token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Config.JWTSecret))
	if e != nil {
		s.resultError(w, e)
		return
	}
	stdhttp.SetCookie(w, &stdhttp.Cookie{Name: "ea_session", Value: token, Path: "/", HttpOnly: true, Secure: s.Config.Env == "production", SameSite: stdhttp.SameSiteLaxMode, MaxAge: 8 * 3600})
	write(w, 200, u)
}
func (s *Server) signup(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if e := decode(w, r, &in); e != nil {
		s.resultError(w, e)
		return
	}
	u, e := s.Service.Signup(r.Context(), in.Name, in.Email, in.Password)
	if e != nil {
		s.resultError(w, e)
		return
	}
	s.session(w, u)
}
func (s *Server) login(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if e := decode(w, r, &in); e != nil {
		s.resultError(w, e)
		return
	}
	u, e := s.Service.Login(r.Context(), in.Email, in.Password)
	if e != nil {
		s.resultError(w, e)
		return
	}
	s.session(w, u)
}
func (s *Server) clearSession(w stdhttp.ResponseWriter) {
	stdhttp.SetCookie(w, &stdhttp.Cookie{Name: "ea_session", Path: "/", HttpOnly: true, Secure: s.Config.Env == "production", SameSite: stdhttp.SameSiteLaxMode, MaxAge: -1})
}
func (s *Server) logout(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	s.clearSession(w)
	w.WriteHeader(204)
}
func number(r *stdhttp.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, e := strconv.Atoi(v)
	if e != nil {
		return -1
	}
	return n
}
func (s *Server) posts(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	v, e := s.Service.Posts(r.Context(), domain.Filter{Page: number(r, "page", 1), Limit: number(r, "limit", 12), Category: r.URL.Query().Get("category"), Tag: r.URL.Query().Get("tag")})
	if e != nil {
		s.resultError(w, e)
		return
	}
	for i := range v.Items {
		v.Items[i].ContentHTML = ""
		v.Items[i].ContentJSON = json.RawMessage("{}")
	}
	write(w, 200, v)
}
func (s *Server) popular(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	v, e := s.Service.Popular(r.Context())
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) suggest(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	v, e := s.Service.Suggest(r.Context(), r.URL.Query().Get("q"))
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) detail(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	slug, e := url.PathUnescape(chi.URLParam(r, "slug"))
	if e != nil || !usecase.ValidSlug(slug) {
		s.resultError(w, domain.ErrNotFound)
		return
	}
	v, e := s.Service.Detail(r.Context(), slug, r.URL.Query().Get("count_view") != "false")
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) comments(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	v, e := s.Service.Comments(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) addComment(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in usecase.CommentInput
	if e := decode(w, r, &in); e != nil {
		s.resultError(w, e)
		return
	}
	v, e := s.Service.AddComment(r.Context(), chi.URLParam(r, "id"), current(r), in)
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 201, v)
}
func (s *Server) taxonomy(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	c, t, e := s.Service.Repo.Taxonomy(r.Context())
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, struct {
		Categories []domain.Taxon `json:"categories"`
		Tags       []domain.Taxon `json:"tags"`
	}{c, t})
}
func (s *Server) ownPosts(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && status != "deleted" {
		s.resultError(w, domain.ErrInvalid)
		return
	}
	v, e := s.Service.StudioPosts(r.Context(), *current(r), status == "deleted")
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) savePost(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in usecase.PostInput
	if e := decode(w, r, &in); e != nil {
		s.resultError(w, e)
		return
	}
	v, e := s.Service.Save(r.Context(), *current(r), in)
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) pending(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "approved" && status != "spam" {
		s.resultError(w, domain.ErrInvalid)
		return
	}
	v, e := s.Service.Repo.ModerationComments(r.Context(), *current(r), status)
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) moderate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if e := decode(w, r, &in); e != nil {
		s.resultError(w, e)
		return
	}
	v, e := s.Service.Moderate(r.Context(), *current(r), chi.URLParam(r, "id"), in.Status)
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) addTaxon(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in domain.Taxon
	if e := decode(w, r, &in); e != nil {
		s.resultError(w, e)
		return
	}
	v, e := s.Service.AddTaxon(r.Context(), *current(r), chi.URLParam(r, "kind"), in)
	if e != nil {
		s.resultError(w, e)
		return
	}
	write(w, 201, v)
}
