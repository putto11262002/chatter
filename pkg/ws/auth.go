package hub

import (
	"fmt"
	"net/http"
)

type Authenticator interface {
	// Authenticate authenticates the request and returns the client id.
	// In the case of a successful authentication, it should return the client id.
	// In the case of a fail authentication, it should return an error.
	// Authenticate should be safe to be called concurrently.
	Authenticate(req *http.Request) (string, error)
}

// QueryAuthenticator is an authenticator that retrieves the client id from the query parameter of the request.
// It does not perform any authentication on the client id.
type QueryAuthenticator struct {
	// queryParam is the query parameter to retrieve the client id.
	queryParam string
}

// Authenticate authenticates the request and returns the client id.
func (qa *QueryAuthenticator) Authenticate(req *http.Request) (string, error) {
	query := req.URL.Query()
	id := query.Get(qa.queryParam)
	if id == "" {
		return "", fmt.Errorf("client id not found")
	}
	return id, nil
}
