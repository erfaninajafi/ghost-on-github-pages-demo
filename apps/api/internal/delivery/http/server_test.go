package http

import (
	"context"
	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/platform"
	"expertsacademy/blog/internal/usecase"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type authRepo struct {
	domain.Repository
	user domain.User
}

func (r authRepo) UserByID(context.Context, string) (domain.User, error) { return r.user, nil }
func serverForTest() *Server {
	return &Server{Config: platform.Config{Origin: "https://blog.expertsacademy.ir", JWTSecret: strings.Repeat("k", 32)}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}
func TestCSRFAndCORS(t *testing.T) {
	s := serverForTest()
	handler := s.security(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for _, tc := range []struct {
		method, origin string
		status         int
	}{{"POST", "", 403}, {"POST", "https://evil.example", 403}, {"POST", s.Config.Origin, 204}, {"GET", "", 204}, {"OPTIONS", s.Config.Origin, 204}} {
		r := httptest.NewRequest(tc.method, "/", nil)
		r.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%+v: %d", tc, w.Code)
		}
	}
}
func TestJWTRejectsWrongMethodAudienceAndExpiry(t *testing.T) {
	s := serverForTest()
	id := uuid.NewString()
	s.Service = &usecase.Service{Repo: authRepo{user: domain.User{ID: id, Role: "author"}}}
	handler := s.authenticate(s.requireAuthor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
	for _, tc := range []struct {
		method jwt.SigningMethod
		aud    string
		expiry time.Time
		status int
	}{{jwt.SigningMethodHS256, "expertsacademy-web", time.Now().Add(time.Hour), 204}, {jwt.SigningMethodHS384, "expertsacademy-web", time.Now().Add(time.Hour), 401}, {jwt.SigningMethodHS256, "wrong", time.Now().Add(time.Hour), 401}, {jwt.SigningMethodHS256, "expertsacademy-web", time.Now().Add(-time.Hour), 401}} {
		claims := jwt.RegisteredClaims{Subject: id, Issuer: "expertsacademy-api", Audience: jwt.ClaimStrings{tc.aud}, ExpiresAt: jwt.NewNumericDate(tc.expiry)}
		token, e := jwt.NewWithClaims(tc.method, claims).SignedString([]byte(s.Config.JWTSecret))
		if e != nil {
			t.Fatal(e)
		}
		r := httptest.NewRequest("GET", "/api/v1/studio/posts", nil)
		r.AddCookie(&http.Cookie{Name: "ea_session", Value: token})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("expected %d got %d", tc.status, w.Code)
		}
	}
}
func TestJSONRejectsUnknownAndMultipleValues(t *testing.T) {
	for _, body := range []string{`{"unknown":true}`, `{"name":"a"} {"name":"b"}`} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		var v struct {
			Name string `json:"name"`
		}
		if decode(httptest.NewRecorder(), r, &v) == nil {
			t.Fatal("accepted malformed request", body)
		}
	}
}
func TestForwardedAddressRequiresTrustedPeer(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "172.30.58.0/24")
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.4:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if clientIP(r) != "203.0.113.4" {
		t.Fatal("trusted attacker header")
	}
	r.RemoteAddr = "172.30.58.5:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.4")
	if clientIP(r) != "203.0.113.4" {
		t.Fatal("wrong proxy address")
	}
}

func TestInvalidCookieDoesNotBlockPublicReads(t *testing.T) {
	s := serverForTest()
	handler := s.authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for _, tc := range []struct {
		method, path string
		bearer       bool
		want         int
	}{
		{"GET", "/api/v1/posts", false, 204},
		{"POST", "/api/v1/posts/test/comments", false, 401},
		{"GET", "/api/v1/studio/posts", false, 401},
		{"GET", "/api/v1/auth/me", false, 401},
		{"GET", "/api/v1/posts", true, 401},
	} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		if tc.bearer {
			r.Header.Set("Authorization", "Bearer invalid-token")
		} else {
			r.AddCookie(&http.Cookie{Name: "ea_session", Value: "invalid-token"})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%+v: got %d", tc, w.Code)
		}
		if !tc.bearer {
			cookies := w.Result().Cookies()
			if len(cookies) != 1 || cookies[0].MaxAge != -1 {
				t.Fatal("invalid cookie was not cleared")
			}
		}
	}
}

func (r authRepo) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	u.ID = uuid.NewString()
	return u, nil
}

func TestSignupAcceptsOnlyEmailAndPassword(t *testing.T) {
	s := serverForTest()
	s.Service = &usecase.Service{Repo: authRepo{}}
	r := httptest.NewRequest("POST", "/api/v1/auth/signup", strings.NewReader(`{"email":"new-writer@example.com","password":"correct-password-123"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.signup(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"role":"author"`) || len(w.Result().Cookies()) != 1 {
		t.Fatalf("signup failed: %d %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest("POST", "/api/v1/auth/signup", strings.NewReader(`{"email":"new-writer@example.com","password":"correct-password-123","role":"admin"}`))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.signup(w, r)
	if w.Code != 400 {
		t.Fatal("client supplied role accepted", w.Code)
	}
}
