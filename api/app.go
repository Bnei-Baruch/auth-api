package api

import (
	"context"
	"net/http"

	"github.com/Nerzal/gocloak/v5"
	"github.com/coreos/go-oidc"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	"github.com/Bnei-Baruch/auth-api/pkg/middleware"
)

type App struct {
	Router        *mux.Router
	Handler       http.Handler
	tokenVerifier *oidc.IDTokenVerifier
	client        gocloak.GoCloak
	token         *gocloak.JWT
}

func (a *App) Initialize(authUrl string, accountsUrl string, skipAuth bool, clientID string, clientSecret string) {
	log.Info().Msg("initializing app")

	a.InitializeWithDB(accountsUrl, skipAuth)
	a.initAuthClient(authUrl, clientID, clientSecret)
}

func (a *App) InitializeWithDB(accountsUrl string, skipAuth bool) {

	a.Router = mux.NewRouter()
	a.initializeRoutes()

	if !skipAuth {
		a.initOidc(accountsUrl)
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization"},
	})

	a.Handler = middleware.ContextMiddleware(
		middleware.LoggingMiddleware(
			middleware.RecoveryMiddleware(
				middleware.RealIPMiddleware(
					corsMiddleware.Handler(
						middleware.AuthenticationMiddleware(a.tokenVerifier, skipAuth)(
							a.Router))))))
}

func (a *App) initOidc(issuer string) {
	oidcProvider, err := oidc.NewProvider(context.TODO(), issuer)
	if err != nil {
		log.Fatal().Err(err).Msg("oidc.NewProvider")
	}

	a.tokenVerifier = oidcProvider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
}

func (a *App) initAuthClient(issuer string, u string, p string) {
	client := gocloak.NewClient(issuer)
	//token, err := client.LoginAdmin(u, p, "master")
	token, err := client.LoginClient(u, p, "master")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed init auth client")
	}
	a.client = client
	a.token = token
}

func (a *App) Run(listenAddr string) {
	addr := listenAddr
	if addr == "" {
		addr = ":8080"
	}

	log.Info().Msgf("app run %s", addr)
	if err := http.ListenAndServe(addr, a.Handler); err != nil {
		log.Fatal().Err(err).Msg("http.ListenAndServe")
	}
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/groups", a.getGroups).Methods("GET")
	a.Router.HandleFunc("/users", a.getUsers).Methods("GET")
	a.Router.HandleFunc("/vusers", a.getVerifyUsers).Methods("GET")
	a.Router.HandleFunc("/my_info", a.getMyInfo).Methods("GET")
	a.Router.HandleFunc("/user/{id}", a.getUserByID).Methods("GET")
	a.Router.HandleFunc("/users/{id}", a.getGroupUsers).Methods("GET")
	a.Router.HandleFunc("/request", a.setRequest).Methods("GET")
	a.Router.HandleFunc("/verify", a.verifyUser).Methods("GET")
	a.Router.HandleFunc("/pending", a.setPending).Methods("GET")
	a.Router.HandleFunc("/approve/{id}", a.approveUserByID).Methods("GET")
	a.Router.HandleFunc("/approve", a.approveUser).Methods("GET")
	a.Router.HandleFunc("/remove/{id}", a.removeUser).Methods("GET")
	a.Router.HandleFunc("/cleanup", a.cleanUsers).Methods("GET")
}
