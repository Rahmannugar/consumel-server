//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Rahmannugar/authlier"
	"github.com/Rahmannugar/authlier/emailverification"
	authlierpostgres "github.com/Rahmannugar/authlier/storage/postgres"
	authlierredis "github.com/Rahmannugar/authlier/storage/redis"
	consumelauthentication "github.com/Rahmannugar/consumel-server/internal/authentication"
	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/authentication"
	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	organizationrepositories "github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testOrigin = "https://app.consumel.test"

func TestSignupOTPCreatesConsumelUserAndSession(t *testing.T) {
	app := newAuthenticationTestApp(t)

	cookie, message := app.signUpAndVerify(t, "owner@example.com")
	if len(message.Code) != 6 {
		t.Fatalf("verification code length = %d, want 6", len(message.Code))
	}
	if !cookie.HttpOnly || cookie.Name != "consumel_session" {
		t.Fatalf("verification cookie = %#v, want HttpOnly Consumel session", cookie)
	}

	reusedVerification := performJSONRequest(
		t,
		app.router,
		http.MethodPost,
		"/auth/verify-email",
		map[string]string{"email": "owner@example.com", "code": message.Code},
		nil,
	)
	if reusedVerification.Code != http.StatusBadRequest {
		t.Fatalf(
			"reused verification status = %d, want 400; body = %s",
			reusedVerification.Code,
			reusedVerification.Body.String(),
		)
	}

	contextResponse := performJSONRequest(
		t,
		app.router,
		http.MethodGet,
		"/account",
		nil,
		cookie,
	)
	if contextResponse.Code != http.StatusOK {
		t.Fatalf("context status = %d, body = %s", contextResponse.Code, contextResponse.Body.String())
	}
	assertOrganizationCount(t, contextResponse, 0)

	if _, err := app.users.UserByAuthlierSubjectID(t.Context(), message.UserID); err != nil {
		t.Fatalf("load resolved Consumel user: %v", err)
	}
	var storedInPostgres int
	if err := app.pool.QueryRow(t.Context(), "SELECT count(*) FROM authlier_email_verifications").Scan(&storedInPostgres); err != nil {
		t.Fatalf("count PostgreSQL verification challenges: %v", err)
	}
	if storedInPostgres != 0 {
		t.Fatalf("PostgreSQL verification challenge count = %d, want 0", storedInPostgres)
	}
}

func TestFourthSessionRevokesOldestAndSessionsSurviveRedisOutage(t *testing.T) {
	app := newAuthenticationTestApp(t)
	oldestCookie, _ := app.signUpAndVerify(t, "sessions@example.com")

	// Resolve once so the oldest session is present in Redis before revocation.
	firstContext := performJSONRequest(t, app.router, http.MethodGet, "/account", nil, oldestCookie)
	if firstContext.Code != http.StatusOK {
		t.Fatalf("initial context status = %d, body = %s", firstContext.Code, firstContext.Body.String())
	}

	newestCookie := oldestCookie
	for signInNumber := 1; signInNumber <= 3; signInNumber++ {
		signIn := performJSONRequest(t, app.router, http.MethodPost, "/auth/sign-in", map[string]string{
			"email": "sessions@example.com", "password": "correct horse battery staple",
		}, nil)
		if signIn.Code != http.StatusOK {
			t.Fatalf("sign-in %d status = %d, body = %s", signInNumber, signIn.Code, signIn.Body.String())
		}
		newestCookie = onlySessionCookie(t, signIn)
	}

	revokedContext := performJSONRequest(t, app.router, http.MethodGet, "/account", nil, oldestCookie)
	if revokedContext.Code != http.StatusUnauthorized {
		t.Fatalf("oldest session status = %d, want 401; body = %s", revokedContext.Code, revokedContext.Body.String())
	}
	activeSessions := performJSONRequest(t, app.router, http.MethodGet, "/auth/list-sessions", nil, newestCookie)
	if activeSessions.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d, body = %s", activeSessions.Code, activeSessions.Body.String())
	}
	assertActiveSessionCount(t, activeSessions, 3)

	if err := app.redisContainer.Stop(t.Context(), nil); err != nil {
		t.Fatalf("stop Redis: %v", err)
	}
	postgresContext := performJSONRequest(t, app.router, http.MethodGet, "/account", nil, newestCookie)
	if postgresContext.Code != http.StatusOK {
		t.Fatalf("PostgreSQL fallback status = %d, body = %s", postgresContext.Code, postgresContext.Body.String())
	}
}

func TestSignInRateLimitUsesNormalizedEmail(t *testing.T) {
	app := newAuthenticationTestApp(t)

	for attemptNumber := 1; attemptNumber <= 6; attemptNumber++ {
		failedSignIn := performJSONRequest(t, app.router, http.MethodPost, "/auth/sign-in", map[string]string{
			"email": "  UNKNOWN@example.com ", "password": "not the password",
		}, nil)
		expectedStatus := http.StatusUnauthorized
		if attemptNumber == 6 {
			expectedStatus = http.StatusTooManyRequests
		}
		if failedSignIn.Code != expectedStatus {
			t.Fatalf(
				"failed sign-in %d status = %d, want %d; body = %s",
				attemptNumber,
				failedSignIn.Code,
				expectedStatus,
				failedSignIn.Body.String(),
			)
		}
	}
}

type authenticationTestApp struct {
	router         http.Handler
	sender         *capturingVerificationSender
	users          *userrepositories.UserRepository
	pool           *pgxpool.Pool
	redisContainer testcontainers.Container
}

func newAuthenticationTestApp(t *testing.T) authenticationTestApp {
	t.Helper()
	pool := testdb.OpenMigratedDatabase(t)
	redisContainer, redisClient := openRedis(t)

	postgresDatabase, err := authlierpostgres.New(pool, authlierpostgres.Config{})
	if err != nil {
		t.Fatalf("create Authlier PostgreSQL adapter: %v", err)
	}
	if err := postgresDatabase.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate Authlier database: %v", err)
	}
	redisSessionCache, err := authlierredis.NewSessionCache(redisClient, "consumel-test:auth")
	if err != nil {
		t.Fatalf("create session cache: %v", err)
	}
	sessionCache, err := authentication.NewJitteredSessionCache(redisSessionCache)
	if err != nil {
		t.Fatalf("create jittered session cache: %v", err)
	}
	secret := bytes.Repeat([]byte{0x42}, 32)
	verificationStore, err := authentication.NewRedisEmailVerificationStore(
		pool,
		redisClient,
		secret,
		"consumel-test:email-verification",
	)
	if err != nil {
		t.Fatalf("create Redis email verification store: %v", err)
	}
	distributedLimiter, err := ratelimit.NewRedisLimiter(
		redisClient,
		secret,
		"consumel-test:rate-limit",
	)
	if err != nil {
		t.Fatalf("create distributed rate limiter: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	database, err := authentication.NewAuthlierDatabase(
		postgresDatabase,
		pool,
		sessionCache,
		verificationStore,
		3,
		logger,
	)
	if err != nil {
		t.Fatalf("create Consumel Authlier database: %v", err)
	}
	sender := &capturingVerificationSender{}
	auth, err := authlier.New(authlier.Config{
		AppName:         "Consumel",
		BaseURL:         "https://api.consumel.test",
		BasePath:        "/auth",
		AccountBasePath: "/account",
		Database:        database,
		TrustedOrigins:  []string{testOrigin},
		EmailAndPassword: authlier.EmailAndPasswordConfig{
			Enabled:                  true,
			RequireEmailVerification: true,
			ValidatePassword:         consumelauthentication.ValidatePassword,
			AttemptGuard:             authentication.NewPasswordAttemptGuard(distributedLimiter),
		},
		EmailVerification: authlier.EmailVerificationConfig{
			Enabled:                     true,
			Delivery:                    emailverification.DeliveryMethodOTP,
			OTPSecret:                   secret,
			Sender:                      sender,
			SendOnSignUp:                true,
			AutoSignInAfterVerification: true,
			AttemptGuard:                authentication.NewOTPAttemptGuard(distributedLimiter),
		},
		Session: authlier.SessionConfig{
			Mode:     authlier.SessionModeCookie,
			Lifetime: 7 * 24 * time.Hour,
			Cache:    sessionCache,
			CacheTTL: time.Hour,
			Cookie:   authlier.CookieConfig{Name: "consumel_session", Path: "/"},
		},
	})
	if err != nil {
		t.Fatalf("configure Authlier: %v", err)
	}

	userRepository := userrepositories.NewUserRepository(pool)
	tenantService := authenticationservices.NewAuthenticatedTenantService(
		authentication.NewAuthlierSessionResolver(auth),
		userservices.NewUserService(userRepository),
		organizationrepositories.NewOrganizationRepository(pool),
	)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authenticationhandlers.RegisterRoutes(
		router,
		auth.Handler(),
		tenantService,
		distributedLimiter,
		logger,
	)

	return authenticationTestApp{
		router:         router,
		sender:         sender,
		users:          userRepository,
		pool:           pool,
		redisContainer: redisContainer,
	}
}

func (app authenticationTestApp) signUpAndVerify(
	t *testing.T,
	email string,
) (*http.Cookie, emailverification.Message) {
	t.Helper()
	signUp := performJSONRequest(t, app.router, http.MethodPost, "/auth/sign-up", map[string]string{
		"email": email, "password": "correct horse battery staple",
	}, nil)
	if signUp.Code != http.StatusCreated {
		t.Fatalf("sign-up status = %d, body = %s", signUp.Code, signUp.Body.String())
	}
	message := app.sender.last(t)
	verification := performJSONRequest(t, app.router, http.MethodPost, "/auth/verify-email", map[string]string{
		"email": email, "code": message.Code,
	}, nil)
	if verification.Code != http.StatusOK {
		t.Fatalf("verification status = %d, body = %s", verification.Code, verification.Body.String())
	}
	return onlySessionCookie(t, verification), message
}

func onlySessionCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("session cookies = %#v, want one", cookies)
	}
	return cookies[0]
}

type capturingVerificationSender struct {
	mu       sync.Mutex
	messages []emailverification.Message
}

func (sender *capturingVerificationSender) SendVerification(
	_ context.Context,
	message emailverification.Message,
) error {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.messages = append(sender.messages, message)
	return nil
}

func (sender *capturingVerificationSender) last(t *testing.T) emailverification.Message {
	t.Helper()
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.messages) == 0 {
		t.Fatal("no verification email was sent")
	}
	return sender.messages[len(sender.messages)-1]
}

func openRedis(t *testing.T) (testcontainers.Container, *redis.Client) {
	t.Helper()
	container, err := testcontainers.Run(
		t.Context(),
		"redis:8-alpine",
		testcontainers.WithExposedPorts("6379/tcp"),
		testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections")),
	)
	if err != nil {
		t.Fatalf("start Redis container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate Redis container: %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("get Redis host: %v", err)
	}
	port, err := container.MappedPort(t.Context(), "6379/tcp")
	if err != nil {
		t.Fatalf("get Redis port: %v", err)
	}
	client := redis.NewClient(&redis.Options{
		Addr:         net.JoinHostPort(host, port.Port()),
		DialTimeout:  250 * time.Millisecond,
		ReadTimeout:  250 * time.Millisecond,
		WriteTimeout: 250 * time.Millisecond,
		MaxRetries:   0,
	})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})
	if err := client.Ping(t.Context()).Err(); err != nil {
		t.Fatalf("ping Redis: %v", err)
	}
	return container, client
}

func performJSONRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body any,
	cookie *http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, "https://api.consumel.test"+path, requestBody)
	request.Header.Set("Origin", testOrigin)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertOrganizationCount(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	var body struct {
		Organizations []json.RawMessage `json:"organizations"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode context response: %v", err)
	}
	if len(body.Organizations) != want {
		t.Fatalf("organization count = %d, want %d; body = %s", len(body.Organizations), want, response.Body.String())
	}
}

func assertActiveSessionCount(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	var body struct {
		Sessions []json.RawMessage `json:"sessions"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode sessions response: %v", err)
	}
	if len(body.Sessions) != want {
		t.Fatalf("active session count = %d, want %d; body = %s", len(body.Sessions), want, response.Body.String())
	}
}
