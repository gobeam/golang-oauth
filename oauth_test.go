package go_oauth2

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"testing"
	"time"
)

var dbStore *Store

var accessTokenString string
var refreshTokenString string
var accessId uuid.UUID
var userID int64 = 1

func init() {
	store := NewDefaultStore(
		NewConfig("root:root@tcp(127.0.0.1:8889)/goauth?charset=utf8&parseTime=True&loc=Local"),
	)
	dbStore = store
	//defer store.Close()
}

func TestCreateClient(t *testing.T) {
	client, err := dbStore.CreateClient(1)
	if err != nil {
		t.Error(err.Error())
	}
	if client.ID == uuid.Nil {
		t.Errorf("Client uuid is not expected to be %s", uuid.Nil)
	}
}

func TestCreate(t *testing.T) {
	accessToken := &Token{
		ClientID:        uuid.MustParse("17d5a915-c403-487e-b41f-92fd1074bd30"),
		ClientSecret:    "UnCMSiJqxFg1O7cqL0MM",
		UserID:          userID,
		Scope:           "*",
		AccessCreateAt:  time.Now(),
		AccessExpiresIn: time.Second * 15,
		RefreshCreateAt: time.Now(),
	}
	resp, err := dbStore.Create(accessToken)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.RefreshToken == "" {
		t.Error("refresh token cannot be nil")
	}
	refreshTokenString = resp.RefreshToken
	if resp.AccessToken == "" {
		t.Error("access token cannot be nil")
	}
	accessTokenString = resp.AccessToken
}


func TestGetByAccess(t *testing.T) {

	resp, err := dbStore.GetByAccess(accessTokenString)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.ID == uuid.Nil {
		t.Errorf("token info uuid is not expected to be %s", uuid.Nil)
	}
}

func TestGetByRefresh(t *testing.T) {

	resp, err := dbStore.GetByRefresh(refreshTokenString)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.ID == uuid.Nil {
		t.Errorf("token info uuid is not expected to be %s", uuid.Nil)
	}
	accessId = resp.AccessTokenId
}

func TestRevokeByAccessTokens(t *testing.T) {
	err := dbStore.RevokeByAccessTokens(userID)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRevokeRefreshToken(t *testing.T) {
	err := dbStore.RevokeRefreshToken(accessId.String())
	if err != nil {
		t.Error(err.Error())
	}
}

func TestClearByAccessToken(t *testing.T) {
	err := dbStore.ClearByAccessToken(userID)
	if err != nil {
		t.Error(err.Error())
	}
}

