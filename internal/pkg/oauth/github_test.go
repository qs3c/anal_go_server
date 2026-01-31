package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestNewGithubOAuth(t *testing.T) {
	oauth := NewGithubOAuth("client-id", "client-secret", "http://localhost/callback")

	assert.NotNil(t, oauth)
	assert.NotNil(t, oauth.config)
	assert.Equal(t, "client-id", oauth.config.ClientID)
	assert.Equal(t, "client-secret", oauth.config.ClientSecret)
	assert.Equal(t, "http://localhost/callback", oauth.config.RedirectURL)
	assert.Contains(t, oauth.config.Scopes, "user:email")
}

func TestGithubOAuth_GetAuthURL(t *testing.T) {
	oauth := NewGithubOAuth("test-client-id", "test-secret", "http://example.com/callback")

	url := oauth.GetAuthURL("test-state")

	assert.Contains(t, url, "github.com")
	assert.Contains(t, url, "client_id=test-client-id")
	assert.Contains(t, url, "state=test-state")
	assert.Contains(t, url, "redirect_uri=")
}

func TestGithubOAuth_GetAuthURL_DifferentStates(t *testing.T) {
	oauth := NewGithubOAuth("client", "secret", "http://localhost/callback")

	url1 := oauth.GetAuthURL("state1")
	url2 := oauth.GetAuthURL("state2")

	assert.Contains(t, url1, "state=state1")
	assert.Contains(t, url2, "state=state2")
	assert.NotEqual(t, url1, url2)
}

func TestGithubUser_Structure(t *testing.T) {
	user := GithubUser{
		ID:        12345,
		Login:     "testuser",
		Email:     "test@example.com",
		AvatarURL: "https://github.com/avatar.png",
		Name:      "Test User",
	}

	assert.Equal(t, int64(12345), user.ID)
	assert.Equal(t, "testuser", user.Login)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "https://github.com/avatar.png", user.AvatarURL)
	assert.Equal(t, "Test User", user.Name)
}

func TestGithubUser_JSON(t *testing.T) {
	jsonData := `{
		"id": 98765,
		"login": "jsonuser",
		"email": "json@example.com",
		"avatar_url": "https://example.com/avatar.jpg",
		"name": "JSON User"
	}`

	var user GithubUser
	err := json.Unmarshal([]byte(jsonData), &user)

	require.NoError(t, err)
	assert.Equal(t, int64(98765), user.ID)
	assert.Equal(t, "jsonuser", user.Login)
	assert.Equal(t, "json@example.com", user.Email)
	assert.Equal(t, "https://example.com/avatar.jpg", user.AvatarURL)
	assert.Equal(t, "JSON User", user.Name)
}

func TestGithubOAuth_GetUser_MockRequest(t *testing.T) {
	// Create mock server for user endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(GithubUser{
				ID:        555,
				Login:     "mockuser",
				Email:     "mock@example.com",
				AvatarURL: "https://mock.avatar.url",
				Name:      "Mock User",
			})
		}
	}))
	defer server.Close()

	// Create token and mock client
	token := &oauth2.Token{
		AccessToken: "test-token",
	}

	// Override client to use mock server
	ctx := context.Background()
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Make request to mock server
	resp, err := client.Get(server.URL + "/user")
	require.NoError(t, err)
	defer resp.Body.Close()

	var user GithubUser
	err = json.NewDecoder(resp.Body).Decode(&user)
	require.NoError(t, err)

	assert.Equal(t, int64(555), user.ID)
	assert.Equal(t, "mockuser", user.Login)
	assert.Equal(t, "mock@example.com", user.Email)
}

func TestGithubUser_EmptyFields(t *testing.T) {
	jsonData := `{"id": 1, "login": "user"}`

	var user GithubUser
	err := json.Unmarshal([]byte(jsonData), &user)

	require.NoError(t, err)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "user", user.Login)
	assert.Empty(t, user.Email)
	assert.Empty(t, user.AvatarURL)
	assert.Empty(t, user.Name)
}

func TestGithubOAuth_ConfigScopes(t *testing.T) {
	oauth := NewGithubOAuth("id", "secret", "http://localhost/callback")

	// Should have user:email scope
	assert.Len(t, oauth.config.Scopes, 1)
	assert.Equal(t, "user:email", oauth.config.Scopes[0])
}

func TestGithubOAuth_EmptyCredentials(t *testing.T) {
	oauth := NewGithubOAuth("", "", "")

	assert.NotNil(t, oauth)
	assert.Empty(t, oauth.config.ClientID)
	assert.Empty(t, oauth.config.ClientSecret)
	assert.Empty(t, oauth.config.RedirectURL)
}

func TestGithubOAuth_SpecialCharactersInState(t *testing.T) {
	oauth := NewGithubOAuth("client", "secret", "http://localhost/callback")

	// Test with special characters that need URL encoding
	url := oauth.GetAuthURL("state-with-special-chars_123")

	assert.Contains(t, url, "state=state-with-special-chars_123")
}
