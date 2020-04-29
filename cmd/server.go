package cmd

import (
	"os"

	"github.com/Bnei-Baruch/auth-api/api"
)

func Init() {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	accountsUrl := os.Getenv("ACC_URL")
	authUrl := os.Getenv("AUTH_URL")
	//authUser := os.Getenv("AUTH_USER")
	//authPass := os.Getenv("AUTH_PASS")
	clientID := os.Getenv("AUTH_ID")
	cleintSecret := os.Getenv("AUTH_SECRET")
	skipAuth := os.Getenv("SKIP_AUTH") == "true"

	a := api.App{}
	a.Initialize(authUrl, accountsUrl, skipAuth, clientID, cleintSecret)
	a.Run(listenAddress)
}

