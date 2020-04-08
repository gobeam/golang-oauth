package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	goOauth2 "github.com/gobeam/golang-oauth"
	"github.com/gobeam/golang-oauth/example/common"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/gobeam/golang-oauth/example/routers"
	"log"
)

func main() {

	dbUrl := common.GetConfig("mysql", "url").String()
	db, err := gorm.Open("mysql", dbUrl)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	store := goOauth2.NewDefaultStore(
		goOauth2.NewConfig(dbUrl),
	)
	defer store.Close()
	models.InitializeDb(db.Debug())

	// register custom validator
	newValidator := common.NewValidatorRegister(db)
	newValidator.RegisterValidator()

	// router setup
	router := routers.SetupRouter(store)

	serverError := router.Run(fmt.Sprintf(":%s", common.GetConfig("system", "httpport").String()))
	if serverError != nil {
		log.Fatalf("Server failed to start %v ", serverError)
	}
}

