package api

import (
	"context"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"net/http"
	"os"

	"github.com/Nerzal/gocloak/v11"
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
	Msg           mqtt.Client
}

func (a *App) Initialize(authUrl string, accountsUrl string, skipAuth bool, clientID string, clientSecret string) {
	log.Info().Msg("initializing app")

	a.InitApp(accountsUrl, skipAuth)
	a.initAuthClient(authUrl, clientID, clientSecret)
}

func (a *App) InitApp(accountsUrl string, skipAuth bool) {

	a.Router = mux.NewRouter()
	a.initializeRoutes()
	a.initMQTT()

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
	ctx := context.Background()
	//token, err := client.LoginAdmin(ctx, u, p, "master")
	token, err := client.LoginClient(ctx, u, p, "master")
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
	a.Router.HandleFunc("/reg_check", a.regCheck).Methods("GET")
	a.Router.HandleFunc("/my_info", a.getMyInfo).Methods("GET")
	a.Router.HandleFunc("/find", a.findUser).Methods("GET")
	a.Router.HandleFunc("/user/{id}", a.getUserInfo).Methods("GET")
	a.Router.HandleFunc("/search", a.searchUsers).Methods("GET")
	a.Router.HandleFunc("/users/{id}", a.getGroupUsers).Methods("GET")
	a.Router.HandleFunc("/request", a.setRequest).Methods("GET")
	a.Router.HandleFunc("/verify", a.verifyUser).Methods("GET")
	a.Router.HandleFunc("/pending", a.setPending).Methods("POST")
	a.Router.HandleFunc("/status", a.changeStatus).Methods("POST")
	a.Router.HandleFunc("/kmedia", a.kmediaGroup).Methods("POST")
	a.Router.HandleFunc("/approve/{id}", a.approveUserByID).Methods("GET")
	a.Router.HandleFunc("/remove/{id}", a.removeUser).Methods("GET")
	a.Router.HandleFunc("/cleanup", a.cleanUsers).Methods("GET")
	a.Router.HandleFunc("/self_remove", a.selfRemove).Methods("DELETE")
}

func (a *App) initMQTT() {
	if os.Getenv("MQTT_URL") != "" {
		server := os.Getenv("MQTT_URL")
		username := os.Getenv("MQTT_USER")
		password := os.Getenv("MQTT_PASS")

		opts := mqtt.NewClientOptions()
		opts.AddBroker(fmt.Sprintf("ssl://%s", server))
		opts.SetClientID("auth_mqtt_client")
		opts.SetUsername(username)
		opts.SetPassword(password)
		opts.SetAutoReconnect(true)
		opts.SetOnConnectHandler(a.SubMQTT)
		opts.SetConnectionLostHandler(a.LostMQTT)
		a.Msg = mqtt.NewClient(opts)
		if token := a.Msg.Connect(); token.Wait() && token.Error() != nil {
			err := token.Error()
			log.Fatal().Err(err).Msg("initialize mqtt listener")
		}
	}
}
