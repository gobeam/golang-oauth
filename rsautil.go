package go_oauth2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	rand2 "math/rand"
	"os"
)

// RandomKeyCharacters is random key characters to choose from
var RandomKeyCharacters = []byte("abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandomKey generates random characters from RandomKeyCharacters
func RandomKey(length int) string {
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		randInt := randInt(0, len(RandomKeyCharacters))
		bytes[i] = RandomKeyCharacters[randInt : randInt+1][0]
	}
	return string(bytes)
}

// randInt generates a random integer between min and max.
func randInt(min int, max int) int {
	return min + rand2.Intn(max-min)
}

// GenerateKeyPair generates a new key pair
func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey) {
	privkey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		log.Fatal(err)
	}
	return privkey, &privkey.PublicKey
}

// SavePEMKey saves generated *rsa.PrivateKey to file
func SavePEMKey(fileName string, key *rsa.PrivateKey) {
	outFile, err := os.Create(fileName)
	checkError(err)
	defer outFile.Close()

	var privateKey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(outFile, privateKey)
	checkError(err)
}

// SavePublicPEMKey saves generated *rsa.PublicKey to file
func SavePublicPEMKey(fileName string, pubkey *rsa.PublicKey) {
	pubASN1, err := x509.MarshalPKIXPublicKey(pubkey)
	checkError(err)

	var pemkey = &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	}

	pemfile, err := os.Create(fileName)
	checkError(err)
	defer pemfile.Close()

	err = pem.Encode(pemfile, pemkey)
	checkError(err)
}

// checkError checks for error
func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

// BytesToPublicKey converts given bytes to *rsa.PublicKey
func BytesToPublicKey(pub []byte) *rsa.PublicKey {
	block, _ := pem.Decode(pub)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		log.Println("is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	ifc, err := x509.ParsePKIXPublicKey(b)
	if err != nil {
		log.Fatal(err)
	}
	key, ok := ifc.(*rsa.PublicKey)
	if !ok {
		log.Fatal("not ok")
	}
	return key
}

// BytesToPrivateKey converts given bytes to *rsa.PrivateKey
func BytesToPrivateKey(priv []byte) *rsa.PrivateKey {
	block, _ := pem.Decode(priv)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		log.Println("is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		log.Fatal(err)
	}
	return key
}

// EncryptWithPublicKey encrypts given []byte, with public key
func EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) (string, error) {
	label := []byte(Label)
	rng := rand.Reader
	cipherText, err := rsa.EncryptOAEP(sha256.New(), rng, pub, []byte(msg), label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from encryption: %s\n", err)
		return "", err
	}
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// DecryptWithPrivateKey decrypts given []byte, with private key
func DecryptWithPrivateKey(cipherText string, priv *rsa.PrivateKey) (string, error) {
	ct, _ := base64.StdEncoding.DecodeString(cipherText)
	label := []byte(Label)

	// crypto/rand.Reader is a good source of entropy for blinding the RSA
	// operation.
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, priv, ct, label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from decryption: %s\n", err)
		return "", err
	}
	fmt.Printf("Plaintext: %s\n", string(plaintext))

	return string(plaintext), nil
}
