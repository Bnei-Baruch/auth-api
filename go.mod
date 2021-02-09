module github.com/Bnei-Baruch/auth-api

go 1.14

require (
	github.com/Nerzal/gocloak/v5 v5.5.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/eclipse/paho.golang v0.9.0
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/go-resty/resty/v2 v2.4.0 // indirect
	github.com/goiiot/libmqtt v0.9.6
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.8.0
	github.com/pkg/errors v0.9.1
	github.com/pquerna/cachecontrol v0.0.0-20201205024021-ac21108117ac // indirect
	github.com/rs/cors v1.7.0
	github.com/rs/zerolog v1.20.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777 // indirect
	golang.org/x/oauth2 v0.0.0-20210201163806-010130855d6c // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
)

replace github.com/eclipse/paho.golang => github.com/edoshor/paho.golang v0.9.1-0.20210102034404-01e231e293df