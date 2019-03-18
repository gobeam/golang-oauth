package main

import (
	"fmt"
	"github.com/roshanr83/go-oauth2/oauth"
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

	store := oauth.NewDefaultStore(
		oauth.NewConfig("root:root@tcp(127.0.0.1:8889)/goauth?charset=utf8&parseTime=True&loc=Local"),
	)
	defer store.Close()


	accessToken := &oauth.Token{
		ClientID:        "1",
		UserID:          "1",
		Scope:           "*",
		AccessCreateAt:  time.Now(),
		AccessExpiresIn: time.Second * 6,
		RefreshCreateAt: time.Now(),
	}
	payload, err := store.Create(accessToken)
	fmt.Println("Tokens:",payload)
	fmt.Println("err:",err)
	//payload, err := oauth.DecryptRefreshToken("txkIafX77jK53jLRfH+YpyZXdD+LVXXOd7h4WLnkv1qLL6ylMIH74atxg/XBWlMRqneQHX5IFaWeNgkEhDDfXtokkO8rJOuXhLA49YYiGy8P5r9ybME9NiCZaTpuOIbe2BjYGmxtKyWZ0O/hRUf/6gPaFoaf/DrBmrg4oPqcOM/F6NcKS6k2WEK3MVM1RnOy8Zo15XZ37ZcTht66BC3TZMAclc2R9XRta87/hulo/G4mGi4FiDoNGU9IC8vrLacnpqgM0rVdj1hbghWxeabkt5qHyJC+08hE22rmEomHkX+wX9pb2uxEeSY/BoU9uqiJh+dNXlS3lJEVUMn6Sn6DCQ==")
	//fmt.Println("Tokens:",payload)
	//fmt.Println("err:",err)
}