package api_test

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"

	"example.com/go-chat/internal/api"
	"example.com/go-chat/pkg/auth"
	"example.com/go-chat/pkg/user"
)

func sendSigninRequest(t *testing.T, uc *UserClient, payload api.SigninPayload) *http.Response {

	body := encodeJsonBody(t, payload)

	url, err := url.JoinPath(uc.Server.URL, "/users/signin")
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Client().Post(url, "application/json", body)
	if err != nil {
		t.Fatal(err)
	}

	return res

}

func sendSignupRequest(t *testing.T, uc *UserClient, payload api.SignupPayload) *http.Response {
	body := encodeJsonBody(t, payload)
	url, err := url.JoinPath(uc.Server.URL, "/users/signup")
	if err != nil {
		t.Fatal(err)
	}

	res, err := uc.Client().Post(url, "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func sendMeRequest(t *testing.T, uc *UserClient) *http.Response {
	url, err := url.JoinPath(uc.Server.URL, "/users/me")
	if err != nil {
		t.Fatal(err)
	}

	res, err := uc.Client().Get(url)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func setUpApi(t *testing.T) (*api.Api, func()) {

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	migrationFS := os.DirFS("../../migrations")
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal(err)
	}

	if err := goose.Up(db, "."); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := api.ApiConfig{
		TokenOptions: auth.TokenOptions{
			Exp:    time.Hour,
			Secret: []byte("secret"),
		},
	}

	return api.NewApi(ctx, db, config), func() {
		db.Close()
	}

}

func Test_SignupHandler(t *testing.T) {

	server, close := setUpTestApiServer(t)
	defer close()

	user := NewUserClient(t, user.User{
		Username: "testuser",
		Password: "password",
		Name:     "Test User",
	}, server)

	t.Run("successfully sign up a new user", func(t *testing.T) {

		res := sendSignupRequest(t, user, api.SignupPayload{
			Username: user.User.Username,
			Password: user.User.Password,
			Name:     user.User.Name,
		})

		assert.Equal(t, http.StatusCreated, res.StatusCode)

	})

	t.Run("failed to sign up due to existing user", func(t *testing.T) {
		res := sendSignupRequest(t, user, api.SignupPayload{
			Username: user.User.Username,
			Password: user.User.Password,
			Name:     user.User.Name,
		})

		assert.Equal(t, http.StatusConflict, res.StatusCode)

		var resBody api.ApiError[interface{}]

		decodeJsonBody(t, res, &resBody)

		assert.Equal(t, http.StatusConflict, resBody.Code)
		assert.Equal(t, "user already exists", resBody.Message)
		assert.Empty(t, resBody.Data)
	})
}

func Test_SigninHandler(t *testing.T) {

	server, close := setUpTestApiServer(t)
	defer close()

	user := NewUserClient(t, user.User{
		Username: "testuser",
		Password: "password",
		Name:     "Test User",
	}, server)

	sendSignupRequest(t, user, api.SignupPayload{
		Username: user.User.Username,
		Password: user.User.Password,
		Name:     user.User.Name,
	})

	t.Run("user does not exist", func(t *testing.T) {
		res := sendSigninRequest(t, user, api.SigninPayload{
			Username: "invalid",
			Password: user.User.Password,
		})

		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var body api.ApiError[interface{}]
		decodeJsonBody(t, res, &body)

		assert.Equal(t, "invalid credentials", body.Message)
		assert.Equal(t, http.StatusUnauthorized, body.Code)
		assert.Empty(t, body.Data)

	})

	t.Run("invalid password", func(t *testing.T) {
		res := sendSigninRequest(t, user, api.SigninPayload{
			Username: user.User.Username,
			Password: "invalid",
		})

		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var body api.ApiError[interface{}]
		decodeJsonBody(t, res, &body)

		assert.Equal(t, "invalid credentials", body.Message)
		assert.Equal(t, http.StatusUnauthorized, body.Code)
		assert.Empty(t, body.Data)
	})

	t.Run("successful signin", func(t *testing.T) {
		res := sendSigninRequest(t, user, api.SigninPayload{
			Username: user.User.Username,
			Password: user.User.Password,
		})

		assert.Equal(t, http.StatusOK, res.StatusCode)

		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var body api.SigninResponse
		decodeJsonBody(t, res, &body)

		assert.NotEmpty(t, body.Token)
		assert.NotEmpty(t, body.ExpireAt)

		var authCookie *http.Cookie
		for _, cookie := range res.Cookies() {
			if cookie.Name == api.AuthCookieName {
				authCookie = cookie
			}
		}

		assert.NotNil(t, authCookie)
		assert.Equal(t, body.Token, authCookie.Value, "token in response and cookie should match")

	})

}

func Test_MeHandler(t *testing.T) {
	server, close := setUpTestApiServer(t)
	defer close()

	user := NewUserClient(t, user.User{
		Username: "testuser",
		Password: "password",
		Name:     "Test User",
	}, server)

	sendSignupRequest(t, user, api.SignupPayload{
		Username: user.User.Username,
		Password: user.User.Password,
		Name:     user.User.Name,
	})

	t.Run("unauthenticated request", func(t *testing.T) {
		res := sendMeRequest(t, user)

		assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var body api.ApiError[interface{}]
		decodeJsonBody(t, res, &body)

		assert.Equal(t, "Unauthenticated", body.Message)
		assert.Equal(t, http.StatusUnauthorized, body.Code)
		assert.Empty(t, body.Data)
	})

	t.Run("authenticated request", func(t *testing.T) {
		sendSigninRequest(t, user, api.SigninPayload{
			Username: user.User.Username,
			Password: user.User.Password,
		})

		res := sendMeRequest(t, user)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var body api.UserResponse

		decodeJsonBody(t, res, &body)

		assert.Equal(t, user.User.Username, body.Username)
		assert.Equal(t, user.User.Name, body.Name)
	})

}
