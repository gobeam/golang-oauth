package main

import (
	"github.com/ekbana/golang-oauth/libs/oauth"
	"gopkg.in/oauth2.v3/manage"
)

func main () {
	//reader := rand.Reader
	//bitSize := 2048
	//
	//key, err := rsa.GenerateKey(reader, bitSize)
	//checkError(err)
	//
	//publicKey := key.PublicKey
	//
	//util.SaveGobKey("private.key", key)
	//util.SavePEMKey("private.pem", key)
	//
	//util.SaveGobKey("public.key", publicKey)
	//util.SavePublicPEMKey("public.pem", publicKey)

	manager := manage.NewDefaultManager()
	NewDefaultStore(
		NewConfig("root:root@tcp(127.0.0.1:8889)/goauth?charset=utf8&parseTime=True&loc=Local"),
	)

	manager.MapTokenStorage(store)
}