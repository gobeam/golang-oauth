package goOauth2

import (
	"crypto/rsa"
	"io/ioutil"
	"os"
	"testing"
)

var publicKeyByte *rsa.PublicKey
var privateKeyByte *rsa.PrivateKey
var message = "Hello World"
var encryptedData string

func TestRandomKey(t *testing.T) {
	var length int = 20
	value := RandomKey(length)
	if value == "" {
		t.Error("value cannot be empty")
	}

	if len(value) < length || len(value) > length {
		t.Errorf("length of value should be %d not %d", length, len(value))
	}
}

func TestGenerateKeyPair(t *testing.T) {
	priv, pub := GenerateKeyPair(BitSize)
	SavePEMKey(PrivatePem, priv)
	SavePublicPEMKey(PublicPem, pub)
	if _, err := os.Stat(PublicPem); os.IsNotExist(err) {
		t.Errorf("dailed to create %s", PublicPem)
	}
	if _, err := os.Stat(PrivatePem); os.IsNotExist(err) {
		t.Errorf("dailed to create %s", PrivatePem)
	}
}

func TestBytesToPublicKey(t *testing.T) {
	pubKeyFile, err := ioutil.ReadFile(PublicPem)
	if err != nil {
		t.Error(err.Error())
	}
	pubkey := BytesToPublicKey(pubKeyFile)
	if err != nil {
		t.Error(err.Error())
	}
	publicKeyByte = pubkey
}

func TestBytesToPrivateKey(t *testing.T) {
	privKey, err := ioutil.ReadFile(PrivatePem)
	if err != nil {
		t.Error(err.Error())
	}
	prikey := BytesToPrivateKey(privKey)
	if err != nil {
		t.Error(err.Error())
	}
	privateKeyByte = prikey
}

func TestEncryptWithPublicKey(t *testing.T) {
	data, err := EncryptWithPublicKey([]byte(message), publicKeyByte)
	if err != nil {
		t.Error(err.Error())
	}
	encryptedData = data
}

func TestDecryptWithPrivateKey(t *testing.T) {
	data, err := DecryptWithPrivateKey(encryptedData, privateKeyByte)
	if err != nil {
		t.Error(err.Error())
	}
	if data != message {
		t.Errorf("Decrypted message should be %s but instead got %s", message, data)
	}
}
