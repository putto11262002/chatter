package api

import (
	"context"
	"database/sql"
	"net/http"

	"example.com/go-chat/pkg/auth"
	"example.com/go-chat/pkg/chat"
	"example.com/go-chat/pkg/user"
	"github.com/go-chi/cors"
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

func (a *Api) mountHandlers() {
	userStore := user.NewSQLiteUserStore(a.db)
	auth := auth.NewSimpleAuth(userStore, a.db, a.config.TokenOptions)
	chat := chat.NewSQLiteChatStore(a.db, userStore)

	userHandler := NewUserHandler(userStore, auth)

	chatHandler := NewChatHandler(chat)

	a.mux.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // TODO: change this to the actual frontend URL
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	a.mux.Route("/users", func(r *ApiMux) {
		r.Post("/signup", userHandler.SignupHandler)
		r.Post("/signin", userHandler.SigninHandler)
		r.Get("/{userID}", userHandler.GetUserByIDHandler)

		r.With(JWTMiddleware(auth)).Get("/me", userHandler.MeHandler)
	})

	a.mux.Route("/chats", func(r *ApiMux) {
		r.Use(JWTMiddleware(auth))
		r.Post("/private", chatHandler.CreatePrivateChatHandler)
		r.Get("/{roomID}", chatHandler.GetRoomByIDHandler)
		r.Get("/me/rooms", chatHandler.GetMyUserRoomsHandler)
		r.Post("/{roomID}/messages", chatHandler.SendMessageToRoomHandler)
		r.Get("/rooms/{roomID}/messages", chatHandler.GetRoomMessagesHandler)
	})

	a.Mux()

}
