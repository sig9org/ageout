package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestParsePrivateKey(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"raw": privateKey,
		"pem": func() []byte {
			der, err := x509.MarshalPKCS8PrivateKey(privateKey)
			if err != nil {
				t.Fatal(err)
			}
			return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := parsePrivateKey(data)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Public().(ed25519.PublicKey).Equal(publicKey) {
				t.Fatal("parsed key has a different public key")
			}
		})
	}
}
