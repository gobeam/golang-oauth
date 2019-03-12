package main

import (
	"github.com/roshanr83/go-oauth2/oauth"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"time"
)

func main () {
	//bitSize := 2048
	//
	//priv, pub := util.GenerateKeyPair(bitSize)
	////checkError(err)
	//
	//
	//util.SaveGobKey("private.key", priv)
	//util.SavePEMKey("private.pem", priv)
	//
	//util.SaveGobKey("public.key", pub)
	//util.SavePublicPEMKey("public.pem", pub)

	manager := manage.NewDefaultManager()
	store := oauth.NewDefaultStore(
		oauth.NewConfig("root:root@tcp(127.0.0.1:8889)/goauth?charset=utf8&parseTime=True&loc=Local"),
	)
	defer store.Close()


	refreshToken := "asdfasdf"
	accessToken := &models.Token{
		ClientID:        "1",
		UserID:          "1",
		Scope:           "*",
		Access:          "Adsdfsdsdf",
		AccessCreateAt:  time.Now(),
		AccessExpiresIn: time.Second * 6,
		Refresh:         refreshToken,
		RefreshCreateAt: time.Now(),
	}
	store.Create(accessToken)

	manager.MapTokenStorage(store)
}