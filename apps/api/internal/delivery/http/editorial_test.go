package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/platform"
	"expertsacademy/blog/internal/usecase"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestAdminMiddlewareChecksRole(t *testing.T) {
	s := serverForTest()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, tc := range []struct {
		role string
		want int
	}{{"", 401}, {"reader", 403}, {"author", 403}, {"admin", 204}} {
		r := httptest.NewRequest("POST", "/api/v1/studio/authors", nil)
		if tc.role != "" {
			r = r.WithContext(context.WithValue(r.Context(), userKey{}, &domain.User{Role: tc.role}))
		}
		w := httptest.NewRecorder()
		s.requireAdmin(next).ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatal(tc, w.Code)
		}
	}
}
func TestAuthorCannotReachAdministrativeRoutes(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL not set")
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	s := serverForTest()
	s.Redis = platform.Redis{Client: redis.NewClient(options)}
	defer s.Redis.Client.Close()
	id := uuid.NewString()
	s.Service = &usecase.Service{Repo: authRepo{user: domain.User{ID: id, Role: "author"}}}
	claims := jwt.RegisteredClaims{Subject: id, Issuer: "expertsacademy-api", Audience: jwt.ClaimStrings{"expertsacademy-web"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	handler := s.Handler()
	for _, tc := range []struct{ method, path string }{
		{"GET", "/studio/authors"}, {"POST", "/studio/authors"}, {"PATCH", "/studio/authors/" + id},
		{"PATCH", "/studio/posts/" + id + "/status"}, {"DELETE", "/studio/posts/" + id},
		{"GET", "/studio/posts?status=deleted"}, {"GET", "/studio/comments"}, {"PATCH", "/studio/comments/" + id},
		{"POST", "/studio/taxonomy/categories"},
	} {
		r := httptest.NewRequest(tc.method, "/api/v1"+tc.path, strings.NewReader(`{}`))
		r.Header.Set("Origin", s.Config.Origin)
		r.Header.Set("Content-Type", "application/json")
		r.AddCookie(&http.Cookie{Name: "ea_session", Value: token})
		r.RemoteAddr = id + ":1" // Isolate the rate-limit bucket from other integration tests.
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
}
func TestDisabledAccountRejectsExistingJWT(t *testing.T) {
	s := serverForTest()
	now := time.Now()
	id := uuid.NewString()
	s.Service = &usecase.Service{Repo: authRepo{user: domain.User{ID: id, Role: "author", DisabledAt: &now}}}
	claims := jwt.RegisteredClaims{Subject: id, Issuer: "expertsacademy-api", Audience: jwt.ClaimStrings{"expertsacademy-web"}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/api/v1/studio/posts", nil)
	r.AddCookie(&http.Cookie{Name: "ea_session", Value: token})
	w := httptest.NewRecorder()
	s.authenticate(s.requireAuthor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))).ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("disabled session remained valid", w.Code)
	}
}
func TestCreatingAuthorDoesNotReplaceAdminSession(t *testing.T) {
	s := serverForTest()
	s.Service = &usecase.Service{Repo: authRepo{}}
	r := httptest.NewRequest("POST", "/api/v1/studio/authors", strings.NewReader(`{"name":"نویسنده جدید","email":"new@example.com","password":"correct-password-123"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), userKey{}, &domain.User{ID: uuid.NewString(), Role: "admin"}))
	w := httptest.NewRecorder()
	s.createAuthor(w, r)
	if w.Code != 201 || len(w.Result().Cookies()) != 0 || !strings.Contains(w.Body.String(), `"role":"author"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
