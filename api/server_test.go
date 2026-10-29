package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aoamusat/hexgogen/configs"
	"github.com/gin-gonic/gin"
)

type recordingLogger struct {
	infos []string
	errs  []error
}

func (l *recordingLogger) Info(msg string)           { l.infos = append(l.infos, msg) }
func (*recordingLogger) Debug(string)                {}
func (*recordingLogger) Warn(string)                 {}
func (l *recordingLogger) Error(_ string, err error) { l.errs = append(l.errs, err) }

func TestRouterMiddleware(t *testing.T) {
	for _, env := range []string{configs.DEVELOPMENT, configs.PRODUCTION} {
		t.Run(env, func(t *testing.T) {
			defer gin.SetMode(gin.TestMode)
			log := &recordingLogger{}
			router := initializeRouter(configs.Config{Env: env}, log)
			router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			router.GET("/panic", func(*gin.Context) { panic("handler failed") })
			for _, tc := range []struct {
				path   string
				status int
			}{
				{"/ok", http.StatusNoContent}, {"/panic", http.StatusInternalServerError},
			} {
				request := httptest.NewRequest(http.MethodGet, tc.path, nil)
				request.Header.Set("Origin", "https://frontend.example.com")
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)
				if recorder.Code != tc.status {
					t.Fatalf("%s: got %d, want %d", tc.path, recorder.Code, tc.status)
				}
				if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Fatal("CORS middleware missing")
				}
				if recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
					t.Fatal("wildcard CORS must not allow credentials")
				}
			}
			if len(log.infos) != 2 {
				t.Fatalf("want one access log per request, got %d", len(log.infos))
			}
			if len(log.errs) != 1 || !strings.Contains(log.errs[0].Error(), "handler failed") {
				t.Fatalf("want one panic error log, got %v", log.errs)
			}
			request := httptest.NewRequest(http.MethodOptions, "/ok", nil)
			request.Header.Set("Origin", "https://frontend.example.com")
			request.Header.Set("Access-Control-Request-Method", "GET")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("preflight status: %d", recorder.Code)
			}
		})
	}
}

type stubHTTPServer struct {
	listen   func() error
	shutdown func(context.Context) error
	close    func() error
}

func (s stubHTTPServer) ListenAndServe() error              { return s.listen() }
func (s stubHTTPServer) Shutdown(ctx context.Context) error { return s.shutdown(ctx) }
func (s stubHTTPServer) Close() error                       { return s.close() }

func awaitServe(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("server did not return promptly")
		return nil
	}
}

func TestServeReturnsListenerFailure(t *testing.T) {
	listenErr := errors.New("address already in use")
	server := stubHTTPServer{listen: func() error { return listenErr }}
	result := make(chan error, 1)
	go func() { result <- serve(context.Background(), server, &recordingLogger{}) }()
	if err := awaitServe(t, result); !errors.Is(err, listenErr) {
		t.Fatalf("got %v, want listener failure", err)
	}
}

func TestServeShutsDownOnCancellation(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	var shutdownCalled bool
	var shutdownDeadline time.Time
	server := stubHTTPServer{
		listen: func() error { close(started); <-stopped; return http.ErrServerClosed },
		shutdown: func(ctx context.Context) error {
			shutdownCalled = true
			shutdownDeadline, _ = ctx.Deadline()
			if err := ctx.Err(); err != nil {
				return err
			}
			close(stopped)
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- serve(ctx, server, &recordingLogger{}) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}
	cancel()
	if err := awaitServe(t, result); err != nil {
		t.Fatal(err)
	}
	if !shutdownCalled || shutdownDeadline.IsZero() || time.Until(shutdownDeadline) > 5*time.Second {
		t.Fatal("shutdown must use a fresh context with a five-second deadline")
	}
}

func TestServeForcesCloseWhenShutdownFails(t *testing.T) {
	shutdownErr := errors.New("shutdown deadline exceeded")
	closeErr := errors.New("close failure")
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	closed := false
	server := stubHTTPServer{
		listen:   func() error { cancel(); <-stopped; return http.ErrServerClosed },
		shutdown: func(context.Context) error { return shutdownErr },
		close:    func() error { closed = true; close(stopped); return closeErr },
	}
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- serve(ctx, server, &recordingLogger{}) }()
	err := awaitServe(t, result)
	if !closed || !errors.Is(err, shutdownErr) || !errors.Is(err, closeErr) {
		t.Fatalf("want forced close and both errors, got closed=%v err=%v", closed, err)
	}
}
