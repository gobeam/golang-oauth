# Golang Oauth 2.0  with JWT custom server with example
[![Build][Build-Status-Image]][Build-Status-Url] [![Go Report Card](https://goreportcard.com/badge/github.com/gobeam/golang-oauth?branch=master)](https://goreportcard.com/report/github.com/gobeam/golang-oauth) [![GoDoc][godoc-image]][godoc-url]

Build your own Golang custom Oauth 2.0 server. This package helps you to develop your own custom oauth2 server. With lots of scaffolding done for you you can easily implement your own logic without any hassle.
<br>
Official docs: [Here](https://godoc.org/github.com/gobeam/golang-oauth)

* [Why?](#why)
* [Example](#example)
* [Installation](#installation)
* [Initialization](#initialization)
* [Create Client](#create-client)
* [Create Access Token](create-access-token)
* [Revoke Access/Refresh Token manually](#revoke-accessrefresh-token-manually)
* [Clear All Access Token Of User](#clear-all-access-token-of-user)
* [Running the tests](#running-the-tests)
* [Contributing](#contributing)
* [License](#license)


## Why
I was trying to make my own modified version of OAUTH2 alongside with JWT server and didn't find any good package so, I made one.  This project is modified version of [go-oauth2/oauth2](https://github.com/go-oauth2/oauth2). since this project didn't meet my requirement .
<br>
This package uses <b>EncryptOAEP</b> which encrypts the given data with <b>RSA-OAEP</b> to encrypt token data. Two separate file <b>private.pem</b> and <b>public.pem</b> file will be created on your root folder which includes respective private and public RSA keys which is used for encryption.
<br>


## Example
For easy scaffold and full working REST API example made with framework [gin-gonic/gin](https://github.com/gin-gonic/gin) is included in  [example](https://github.com/gobeam/golang-oauth/tree/master/example) implementing this package.
[Postman Collection](https://github.com/gobeam/golang-oauth/blob/master/example/third_party/postman_import/postman.json)

## Installation

``` bash
$ go get -u -v github.com/gobeam/golang-oauth
```


## Initialization

Easy to initialize just by:

``` go
package main

import (
	_ "github.com/go-sql-driver/mysql"
	oauth "github.com/roshanr83/go-oauth2"
)

func main() {
	//register store
	store := oauth.NewDefaultStore(
		oauth.NewConfig("root:root@tcp(127.0.0.1:8889)/goauth?charset=utf8&parseTime=True&loc=Local"),
	)
	defer store.Close()
}

```


## Create Client

To create client where 1 is user ID Which will return Oauth Clients struct which include client id and secret which is later used to validate client credentials
	 
```go
 var userId = 1 // to know who created can be 0
 var clientName = "my app" // app name can be empty string
 store.CreateClient(userId, clientName)

```


## Create Access Token
Visit [oauthMiddleware.go](https://github.com/gobeam/golang-oauth/blob/master/example/middlewares/oauthMiddleware.go) to get full example on how to handle creating access token and refresh token. 


## Revoke Access/Refresh Token manually

```go
  /*You can manually revoke access token by passing
  userId which you can get from valid token info */
  store.RevokeByAccessTokens(userId) 
  
  /*You can manually revoke refresh token by passing
  accessTokenId which you can get from valid token info */
  store.RevokeRefreshToken(accessTokenId)

```


## Clear All Access Token Of User

```go
  /* you can also clear all token related to
  user by passing TokenInfo from valid token */
  store.ClearByAccessToken(userId)
```


## Running the tests

Database config is used as "root:root@tcp(127.0.0.1:3306)/goauth?charset=utf8&parseTime=True&loc=Local" in const.go file, You may have to change that configuration according to your system config for successful test.

``` bash
$ go test
```


## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
Please make sure to update tests as appropriate.


## License

Released under the MIT License - see `LICENSE.txt` for details.


[Build-Status-Url]: https://travis-ci.org/gobeam/golang-oauth
[Build-Status-Image]: https://travis-ci.org/gobeam/golang-oauth.svg?branch=master
[godoc-url]: https://pkg.go.dev/github.com/gobeam/golang-oauth?tab=doc
[godoc-image]: https://godoc.org/github.com/gobeam/golang-oauth?status.svg
