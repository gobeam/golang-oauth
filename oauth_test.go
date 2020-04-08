package golang_oauth

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/gobeam/golang-oauth/internal"
	"github.com/gobeam/golang-oauth/util"
	"github.com/google/uuid"
	"testing"
	"time"
)

var dbStore *Store
var accessTokenString string
var refreshTokenString string
var accessId uuid.UUID
var userID int64 = 1
var clientDetail *internal.Clients

func init() {
	store := NewDefaultStore(
		NewConfig(util.DbConfig),
	)
	dbStore = store
}

func TestCreateClient(t *testing.T) {
	client, err := dbStore.CreateClient(1)
	if err != nil {
		t.Error(err.Error())
	}
	clientDetail = &client
	if client.ID == uuid.Nil {
		t.Errorf("Client uuid invalid client not expected to be %s", uuid.Nil)
	}
}

func TestCreate(t *testing.T) {
	accessToken := &internal.Token{
		ClientID:        clientDetail.ID,
		ClientSecret:    clientDetail.Secret,
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
	accessId = resp.ID
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
	defer dbStore.Close()
}
