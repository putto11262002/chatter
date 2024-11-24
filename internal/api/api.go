package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-chi/cors"
	"github.com/putto11262002/chatter/pkg/auth"
	"github.com/putto11262002/chatter/pkg/chat"
	"github.com/putto11262002/chatter/pkg/chat/ws"
	"github.com/putto11262002/chatter/pkg/user"
)

type ApiConfig struct {
	TokenOptions auth.TokenOptions
}

type Api struct {
	db      *sql.DB
	mux     *ApiMux
	context context.Context
	config  ApiConfig
}

func NewApi(ctx context.Context, db *sql.DB, config ApiConfig) *Api {
	api := &Api{
		db:      db,
		mux:     NewAPiRouter(),
		context: ctx,
		config:  config,
	}
	api.mountHandlers()
	return api
}

func (a *Api) Mux() http.Handler {
	return a.mux
}

type HubStore struct {
	chatStore chat.ChatStore
}

type AuthAdapter struct {
}

func (a *AuthAdapter) Authenticate(r *http.Request) (string, error) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		return "", auth.ErrUnauthenticated
	}
	return session.Username, nil
}

func (a *Api) mountHandlers() {
	userStore := user.NewSQLiteUserStore(a.db)
	auth := auth.NewSimpleAuth(userStore, a.db, a.config.TokenOptions)
	chatStore := chat.NewSQLiteChatStore(a.db, userStore)

	userHandler := NewUserHandler(userStore, auth)

	chatHandler := NewChatHandler(chatStore)

	hub := ws.NewChatterHub(a.context, chatStore)
	go hub.Start()
	// defer hub.Close()

	authAdapter := &AuthAdapter{}

	clientFactory := ws.NewHubClientFactory(hub, authAdapter)

	a.mux.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // TODO: change this to the actual frontend URL
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	a.mux.With(JWTMiddleware(auth)).HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		clientFactory.HandleFunc(w, r)
	})

	a.mux.Route("/users", func(r *ApiMux) {
		r.Post("/signup", userHandler.SignupHandler)
		r.Post("/signin", userHandler.SigninHandler)
		r.Get("/{userID}", userHandler.GetUserByIDHandler)

		r.With(JWTMiddleware(auth)).Get("/me", userHandler.MeHandler)
	})

	a.mux.Group(func(r *ApiMux) {
		r.Use(JWTMiddleware(auth))
		r.Post("/rooms/private", chatHandler.CreatePrivateChatHandler)
		r.Post("/rooms/group", chatHandler.CreateGroupChatHandler)
		r.Get("/rooms/{roomID}", chatHandler.GetRoomByIDHandler)
		r.Get("/users/me/rooms", chatHandler.GetMyUserRoomsHandler)
		r.Post("/rooms/{roomID}/messages", chatHandler.SendMessageToRoomHandler)
		r.Get("/rooms/{roomID}/messages", chatHandler.GetRoomMessagesHandler)
	})

	a.Mux()

}
