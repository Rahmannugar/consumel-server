//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Rahmannugar/authlier"
	"github.com/Rahmannugar/authlier/emailverification"
	authlierpostgres "github.com/Rahmannugar/authlier/storage/postgres"
	authlierredis "github.com/Rahmannugar/authlier/storage/redis"
	consumelauthentication "github.com/Rahmannugar/consumel-server/internal/authentication"
	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	authenticationrepositories "github.com/Rahmannugar/consumel-server/internal/authentication/repositories"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/infra/authentication"
	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
	"github.com/Rahmannugar/consumel-server/internal/infra/ratelimit"
	organizationrepositories "github.com/Rahmannugar/consumel-server/internal/organizations/repositories"
	organizationservices "github.com/Rahmannugar/consumel-server/internal/organizations/services"
	userrepositories "github.com/Rahmannugar/consumel-server/internal/users/repositories"
	userservices "github.com/Rahmannugar/consumel-server/internal/users/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	user, err := app.users.UserByAuthlierSubjectID(t.Context(), message.UserID)
	if err != nil {
		t.Fatalf("load resolved Consumel user: %v", err)
	}
	organizationService := organizationservices.NewOrganizationManagementService(
		organizationrepositories.NewOrganizationRepository(app.pool),
	)
	if _, _, err := organizationService.CreateOrganization(
		t.Context(),
		"Consumel Test Organization",
		user.ID,
	); err != nil {
		t.Fatalf("create organization: %v", err)
	}
	contextResponse = performJSONRequest(t, app.router, http.MethodGet, "/account", nil, cookie)
	if contextResponse.Code != http.StatusOK {
		t.Fatalf("context with organization status = %d, body = %s", contextResponse.Code, contextResponse.Body.String())
	}
	assertOrganizationCount(t, contextResponse, 1)
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
			"email": "sessions@example.com", "password": "Correct horse 7!",
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

func TestRepeatedSignupDoesNotRevealVerificationStateAndSignInRecoversVerification(t *testing.T) {
	app := newAuthenticationTestApp(t)
	email := "recovery@example.com"
	credentials := map[string]string{"email": email, "password": "Correct horse 7!"}

	created := performJSONRequest(t, app.router, http.MethodPost, "/auth/sign-up", credentials, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("initial sign-up status = %d, body = %s", created.Code, created.Body.String())
	}
	unverifiedConflict := performJSONRequest(
		t, app.router, http.MethodPost, "/auth/sign-up", credentials, nil,
	)
	if unverifiedConflict.Code != http.StatusConflict {
		t.Fatalf("unverified repeat sign-up status = %d, body = %s", unverifiedConflict.Code, unverifiedConflict.Body.String())
	}

	signIn := performJSONRequest(t, app.router, http.MethodPost, "/auth/sign-in", credentials, nil)
	if signIn.Code != http.StatusForbidden || responseErrorCode(t, signIn) != "email_not_verified" {
		t.Fatalf("unverified sign-in status = %d, body = %s", signIn.Code, signIn.Body.String())
	}
	verification := app.lastVerification(t, email)
	verified := performJSONRequest(t, app.router, http.MethodPost, "/auth/verify-email", map[string]string{
		"email": email, "code": verification.Code,
	}, nil)
	if verified.Code != http.StatusOK {
		t.Fatalf("recovery verification status = %d, body = %s", verified.Code, verified.Body.String())
	}

	verifiedEmail := "verified-recovery@example.com"
	_, _ = app.signUpAndVerify(t, verifiedEmail)
	verifiedConflict := performJSONRequest(t, app.router, http.MethodPost, "/auth/sign-up", map[string]string{
		"email": verifiedEmail, "password": "Correct horse 7!",
	}, nil)
	if verifiedConflict.Code != http.StatusConflict {
		t.Fatalf("verified repeat sign-up status = %d, body = %s", verifiedConflict.Code, verifiedConflict.Body.String())
	}
	if verifiedConflict.Body.String() != unverifiedConflict.Body.String() {
		t.Fatalf("repeat sign-up disclosed verification state: before=%s after=%s",
			unverifiedConflict.Body.String(), verifiedConflict.Body.String())
	}
}

type authenticationTestApp struct {
	router         http.Handler
	emailQueue     *emaildelivery.Queue
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
	emailQueue, err := emaildelivery.NewQueue(pool, secret)
	if err != nil {
		t.Fatalf("create email delivery queue: %v", err)
	}
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
			Sender:                      emailQueue,
			SendOnSignUp:                true,
			SendOnSignIn:                true,
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
		authenticationrepositories.NewAccountContextRepository(pool),
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
		emailQueue:     emailQueue,
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
		"email": email, "password": "Correct horse 7!",
	}, nil)
	if signUp.Code != http.StatusCreated {
		t.Fatalf("sign-up status = %d, body = %s", signUp.Code, signUp.Body.String())
	}
	message := app.lastVerification(t, email)
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

func (app authenticationTestApp) lastVerification(
	t *testing.T,
	email string,
) emailverification.Message {
	t.Helper()
	var deliveryID uuid.UUID
	var nonce []byte
	var ciphertext []byte
	var eventType string
	err := app.pool.QueryRow(t.Context(), `SELECT d.id, d.payload_nonce, d.encrypted_payload, o.event_type
		FROM email_deliveries d
		JOIN outbox_events o ON o.aggregate_id = d.id
		WHERE d.template = 'email_verification'
		ORDER BY d.created_at DESC LIMIT 1`).Scan(&deliveryID, &nonce, &ciphertext, &eventType)
	if err != nil {
		t.Fatalf("load queued verification delivery: %v", err)
	}
	if eventType != emaildelivery.EventTypeQueued {
		t.Fatalf("outbox event type = %q, want %q", eventType, emaildelivery.EventTypeQueued)
	}
	if bytes.Contains(ciphertext, []byte(email)) {
		t.Fatal("queued email payload contains the plaintext recipient")
	}
	payload, err := app.emailQueue.Decrypt(deliveryID, nonce, ciphertext)
	if err != nil {
		t.Fatalf("decrypt queued verification delivery: %v", err)
	}
	if payload.Recipient != email || len(payload.Code) != 6 {
		t.Fatalf("queued verification payload = %#v", payload)
	}
	return emailverification.Message{
		UserID: payload.SubjectID, Email: payload.Recipient,
		Code: payload.Code, ExpiresAt: payload.ExpiresAt,
	}
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
		DialTimeout:  2 * time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
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

func responseErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return body.Error.Code
}
