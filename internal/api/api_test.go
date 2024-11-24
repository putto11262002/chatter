package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/putto11262002/chatter/internal/api"
	"github.com/putto11262002/chatter/pkg/auth"
	"github.com/putto11262002/chatter/pkg/user"
)

type UserClient struct {
	Server *httptest.Server
	Jar    http.CookieJar
	User   user.User
}

func (u *UserClient) Client() *http.Client {
	client := u.Server.Client()
	WithCookirJar(client, u.Jar)
	return client
}

func NewUserClient(t *testing.T, user user.User, server *httptest.Server) *UserClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &UserClient{
		Server: server,
		User:   user,
		Jar:    jar,
	}
}

func NewAuthenticatedUserClient(t *testing.T, user user.User, server *httptest.Server) *UserClient {
	userClient := NewUserClient(t, user, server)

	res := sendSignupRequest(t, userClient, api.SignupPayload{
		Username: user.Username,
		Password: user.Password,
		Name:     user.Name,
	})

	if res.StatusCode != http.StatusCreated {
		t.Fatal("failed to signup user")
	}

	res = sendSigninRequest(t, userClient, api.SigninPayload{
		Username: user.Username,
		Password: user.Password,
	})

	if res.StatusCode != http.StatusOK {
		t.Fatal("failed to signin user")
	}

	return userClient
}

func setUpTestApiServer(t *testing.T) (*httptest.Server, func()) {

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}

	migrationFS := os.DirFS("../../migrations")
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	if err := goose.Up(db, "."); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := api.ApiConfig{
		TokenOptions: auth.TokenOptions{
			Exp:    time.Hour,
			Secret: []byte("secret"),
		},
	}

	_api := api.NewApi(ctx, db, config)

	server := httptest.NewServer(_api.Mux())

	return server, func() {
		db.Close()
		server.Close()
	}
}

func executeRequest(req *http.Request, api *api.Api) *http.Response {
	rr := httptest.NewRecorder()
	api.Mux().ServeHTTP(rr, req)
	return rr.Result()
}

func encodeJsonBody(t *testing.T, body interface{}) io.Reader {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(body)
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

func decodeJsonBody(t *testing.T, res *http.Response, v interface{}) {
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func getCookie(name string, res *http.Response) *http.Cookie {
	for _, cookie := range res.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func WithNewCookirJar(client *http.Client) {
	jar, _ := cookiejar.New(nil)
	client.Jar = jar
}

func WithCookirJar(client *http.Client, jar http.CookieJar) {
	client.Jar = jar
}
