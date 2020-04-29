package api

import (
	"context"
	"net/http"

	"github.com/Nerzal/gocloak/v5"
	"github.com/coreos/go-oidc"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/Bnei-Baruch/auth-api/pkg/middleware"
)

type App struct {
	Router         *mux.Router
	Handler        http.Handler
	tokenVerifier  *oidc.IDTokenVerifier
	client 		   gocloak.GoCloak
	token		   *gocloak.JWT
}

func (a *App) Initialize(authUrl string, accountsUrl string, skipAuth bool, clientID string, cleintSecret string) {
	log.Info().Msg("initializing app")

	a.InitializeWithDB(accountsUrl, skipAuth)
	a.initAuthClient(authUrl, clientID, cleintSecret)
}

func (a *App) InitializeWithDB(accountsUrl string, skipAuth bool) {

	a.Router = mux.NewRouter()
	a.initializeRoutes()

	if !skipAuth {
		a.initOidc(accountsUrl)
	}

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Content-Length", "Accept-Encoding", "Content-Range", "Content-Disposition", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "DELETE", "POST", "PUT", "OPTIONS"})
	cors := handlers.CORS(originsOk, headersOk, methodsOk)

	a.Handler = middleware.ContextMiddleware(
		middleware.LoggingMiddleware(
			middleware.RecoveryMiddleware(
				middleware.RealIPMiddleware(
					middleware.AuthenticationMiddleware(a.tokenVerifier, skipAuth)(
						cors(a.Router))))))
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
	a.Router.HandleFunc("/users/{id}", a.getGroupUsers).Methods("GET")
	a.Router.HandleFunc("/check", a.checkUser).Methods("GET")
	a.Router.HandleFunc("/verify", a.verifyUser).Methods("GET")
}
