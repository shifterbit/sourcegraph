package sourcegraphoperator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc"
	mockrequire "github.com/derision-test/go-mockgen/testutil/require"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/auth"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/auth/providers"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/external/session"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/auth/openidconnect"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/sourcegraph/schema"
)

const (
	testOIDCUser = "testOIDCUser"
	testClientID = "testClientID"
	testIDToken  = "testIDToken"
)

// new OIDCIDServer returns a new running mock OIDC ID provider service. It is
// the caller's responsibility to call Close().
func newOIDCIDServer(t *testing.T, code string, providerConfig *schema.SourcegraphOperatorAuthProvider) (server *httptest.Server, emailPtr *string) {
	s := http.NewServeMux()

	s.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(
			map[string]string{
				"issuer":                 providerConfig.Issuer,
				"authorization_endpoint": providerConfig.Issuer + "/oauth2/v1/authorize",
				"token_endpoint":         providerConfig.Issuer + "/oauth2/v1/token",
				"userinfo_endpoint":      providerConfig.Issuer + "/oauth2/v1/userinfo",
			},
		)
		require.NoError(t, err)
	})
	s.HandleFunc("/oauth2/v1/token", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		values, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, code, values.Get("code"))
		require.Equal(t, "authorization_code", values.Get("grant_type"))

		redirectURI, err := url.QueryUnescape(values.Get("redirect_uri"))
		require.NoError(t, err)
		require.Equal(t, "http://example.com/.auth/callback", redirectURI)

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(
			map[string]any{
				"access_token": "testAccessToken",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"scope":        "openid",
				"id_token":     testIDToken,
			},
		)
		require.NoError(t, err)
	})
	email := "alice@sourcegraph.com"
	s.HandleFunc("/oauth2/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		authzHeader := r.Header.Get("Authorization")
		authzParts := strings.Split(authzHeader, " ")
		require.Len(t, authzParts, 2)
		require.Equal(t, "Bearer", authzParts[0])

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(
			map[string]any{
				"sub":            testOIDCUser,
				"profile":        "This is a profile",
				"email":          email,
				"email_verified": true,
				"picture":        "http://example.com/picture.png",
			},
		)
		require.NoError(t, err)
	})

	auth.MockGetAndSaveUser = func(ctx context.Context, op auth.GetAndSaveUserOp) (userID int32, safeErrMsg string, err error) {
		if op.ExternalAccount.ServiceType == ProviderType &&
			op.ExternalAccount.ServiceID == providerConfig.Issuer &&
			op.ExternalAccount.ClientID == testClientID &&
			op.ExternalAccount.AccountID == testOIDCUser {
			return 123, "", nil
		}
		return 0, "safeErr", errors.Errorf("account %q not found in mock", op.ExternalAccount)
	}
	t.Cleanup(func() {
		auth.MockGetAndSaveUser = nil
	})
	return httptest.NewServer(s), &email
}

func TestMiddleware(t *testing.T) {
	cleanup := session.ResetMockSessionStore(t)
	defer cleanup()

	const testCode = "testCode"
	providerConfig := schema.SourcegraphOperatorAuthProvider{
		ClientID:          testClientID,
		ClientSecret:      "testClientSecret",
		LifecycleDuration: 60,
		Type:              ProviderType,
	}
	oidcIDServer, emailPtr := newOIDCIDServer(t, testCode, &providerConfig)
	defer oidcIDServer.Close()
	providerConfig.Issuer = oidcIDServer.URL

	mockProvider := NewProvider(providerConfig).(*provider)
	providers.MockProviders = []providers.Provider{mockProvider}
	defer func() { providers.MockProviders = nil }()

	t.Run("refresh", func(t *testing.T) {
		err := mockProvider.Refresh(context.Background())
		require.NoError(t, err)
	})

	usersStore := database.NewMockUserStore()
	userExternalAccountsStore := database.NewMockUserExternalAccountsStore()
	userExternalAccountsStore.ListFunc.SetDefaultReturn(
		[]*extsvc.Account{
			{
				AccountSpec: extsvc.AccountSpec{
					ServiceType: ProviderType,
				},
			},
		},
		nil,
	)
	db := database.NewMockDB()
	db.UsersFunc.SetDefaultReturn(usersStore)
	db.UserExternalAccountsFunc.SetDefaultReturn(userExternalAccountsStore)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	authedHandler := http.NewServeMux()
	authedHandler.Handle("/.api/", Middleware(db).API(h))
	authedHandler.Handle("/", Middleware(db).App(h))

	doRequest := func(method, urlStr, body string, cookies []*http.Cookie, authed bool) *http.Response {
		req := httptest.NewRequest(method, urlStr, bytes.NewBufferString(body))
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		if authed {
			req = req.WithContext(actor.WithActor(context.Background(), &actor.Actor{UID: 1}))
		}
		resp := httptest.NewRecorder()
		authedHandler.ServeHTTP(resp, req)
		return resp.Result()
	}

	t.Run("unauthenticated API request should pass through", func(t *testing.T) {
		resp := doRequest(http.MethodGet, "http://example.com/.api/foo", "", nil, false)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("login triggers auth flow", func(t *testing.T) {
		urlStr := fmt.Sprintf("http://example.com%s/login?pc=%s", authPrefix, mockProvider.ConfigID().ID)
		resp := doRequest(http.MethodGet, urlStr, "", nil, false)
		assert.Equal(t, http.StatusFound, resp.StatusCode)

		location := resp.Header.Get("Location")
		wantPrefix := mockProvider.config.Issuer + "/"
		assert.True(t, strings.HasPrefix(location, wantPrefix), "%q does not have prefix %q", location, wantPrefix)

		loginURL, err := url.Parse(location)
		require.NoError(t, err)
		assert.Equal(t, mockProvider.config.ClientID, loginURL.Query().Get("client_id"))
		assert.Equal(t, "http://example.com/.auth/callback", loginURL.Query().Get("redirect_uri"))
		assert.Equal(t, "code", loginURL.Query().Get("response_type"))
		assert.Equal(t, "openid profile email", loginURL.Query().Get("scope"))
	})

	t.Run("callback with bad CSRF should fail", func(t *testing.T) {
		badState := &openidconnect.AuthnState{
			CSRFToken:  "bad",
			Redirect:   "/redirect",
			ProviderID: mockProvider.ConfigID().ID,
		}
		urlStr := fmt.Sprintf("http://example.com/.auth/callback?code=%s&state=%s", testCode, badState.Encode())
		resp := doRequest(http.MethodGet, urlStr, "", nil, false)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("callback with good CSRF should set auth cookie", func(t *testing.T) {
		state := &openidconnect.AuthnState{
			CSRFToken:  "good",
			Redirect:   "/redirect",
			ProviderID: mockProvider.ConfigID().ID,
		}
		openidconnect.MockVerifyIDToken = func(rawIDToken string) *oidc.IDToken {
			require.Equal(t, testIDToken, rawIDToken)
			return &oidc.IDToken{
				Issuer:  oidcIDServer.URL,
				Subject: testOIDCUser,
				Expiry:  time.Now().Add(time.Hour),
				Nonce:   state.Encode(),
			}
		}
		defer func() { openidconnect.MockVerifyIDToken = nil }()

		usersStore.GetByIDFunc.SetDefaultHook(func(_ context.Context, id int32) (*types.User, error) {
			return &types.User{
				ID:        id,
				CreatedAt: time.Now(),
			}, nil
		})
		userExternalAccountsStore.CreateUserAndSaveFunc.SetDefaultHook(func(_ context.Context, user database.NewUser, _ extsvc.AccountSpec, _ extsvc.AccountData) (int32, error) {
			assert.True(t, strings.HasPrefix(user.Username, usernamePrefix), "%q does not have prefix %q", user.Username, usernamePrefix)
			return 1, nil
		})

		urlStr := fmt.Sprintf("http://example.com/.auth/callback?code=%s&state=%s", testCode, state.Encode())
		cookies := []*http.Cookie{
			{
				Name:  stateCookieName,
				Value: state.Encode(),
			},
		}
		resp := doRequest(http.MethodGet, urlStr, "", cookies, false)
		assert.Equal(t, http.StatusFound, resp.StatusCode)
		assert.Equal(t, state.Redirect, resp.Header.Get("Location"))
		mockrequire.Called(t, usersStore.SetIsSiteAdminFunc)
	})

	t.Run("callback with bad email domain should fail", func(t *testing.T) {
		oldEmail := *emailPtr
		*emailPtr = "alice@example.com" // Doesn't match requiredEmailDomain
		defer func() { *emailPtr = oldEmail }()

		state := &openidconnect.AuthnState{
			CSRFToken:  "good",
			Redirect:   "/redirect",
			ProviderID: mockProvider.ConfigID().ID,
		}
		openidconnect.MockVerifyIDToken = func(rawIDToken string) *oidc.IDToken {
			require.Equal(t, testIDToken, rawIDToken)
			return &oidc.IDToken{
				Issuer:  oidcIDServer.URL,
				Subject: testOIDCUser,
				Expiry:  time.Now().Add(time.Hour),
				Nonce:   state.Encode(),
			}
		}
		defer func() { openidconnect.MockVerifyIDToken = nil }()

		urlStr := fmt.Sprintf("http://example.com/.auth/callback?code=%s&state=%s", testCode, state.Encode())
		cookies := []*http.Cookie{
			{
				Name:  stateCookieName,
				Value: state.Encode(),
			},
		}
		resp := doRequest(http.MethodGet, urlStr, "", cookies, false)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("no open redirection", func(t *testing.T) {
		state := &openidconnect.AuthnState{
			CSRFToken:  "good",
			Redirect:   "https://evil.com",
			ProviderID: mockProvider.ConfigID().ID,
		}
		openidconnect.MockVerifyIDToken = func(rawIDToken string) *oidc.IDToken {
			require.Equal(t, testIDToken, rawIDToken)
			return &oidc.IDToken{
				Issuer:  oidcIDServer.URL,
				Subject: testOIDCUser,
				Expiry:  time.Now().Add(time.Hour),
				Nonce:   state.Encode(),
			}
		}
		defer func() { openidconnect.MockVerifyIDToken = nil }()

		usersStore.GetByIDFunc.SetDefaultHook(func(_ context.Context, id int32) (*types.User, error) {
			return &types.User{
				ID:        id,
				CreatedAt: time.Now(),
			}, nil
		})

		urlStr := fmt.Sprintf("http://example.com/.auth/callback?code=%s&state=%s", testCode, state.Encode())
		cookies := []*http.Cookie{
			{
				Name:  stateCookieName,
				Value: state.Encode(),
			},
		}
		resp := doRequest(http.MethodGet, urlStr, "", cookies, false)
		assert.Equal(t, http.StatusFound, resp.StatusCode)
		assert.Equal(t, "/", resp.Header.Get("Location"))
		mockrequire.Called(t, usersStore.SetIsSiteAdminFunc)
	})

	t.Run("lifetime expired", func(t *testing.T) {
		usersStore.GetByIDFunc.SetDefaultHook(func(_ context.Context, id int32) (*types.User, error) {
			return &types.User{
				ID:        id,
				CreatedAt: time.Now().Add(-61 * time.Minute),
			}, nil
		})

		state := &openidconnect.AuthnState{
			CSRFToken:  "good",
			Redirect:   "https://evil.com",
			ProviderID: mockProvider.ConfigID().ID,
		}
		openidconnect.MockVerifyIDToken = func(rawIDToken string) *oidc.IDToken {
			require.Equal(t, testIDToken, rawIDToken)
			return &oidc.IDToken{
				Issuer:  oidcIDServer.URL,
				Subject: testOIDCUser,
				Expiry:  time.Now().Add(time.Hour),
				Nonce:   state.Encode(),
			}
		}
		defer func() { openidconnect.MockVerifyIDToken = nil }()

		urlStr := fmt.Sprintf("http://example.com/.auth/callback?code=%s&state=%s", testCode, state.Encode())
		cookies := []*http.Cookie{
			{
				Name:  stateCookieName,
				Value: state.Encode(),
			},
		}
		resp := doRequest(http.MethodGet, urlStr, "", cookies, false)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "The retrieved user account lifecycle has already expired")
	})
}
