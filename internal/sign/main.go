// Command sign creates an Ed25519 detached signature for a file.
package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
)

func main() {
	keyPath := flag.String("key", "", "Ed25519 private key file (raw 64-byte or PKCS#8 PEM)")
	inPath := flag.String("in", "", "file to sign")
	outPath := flag.String("out", "", "signature output file")
	flag.Parse()
	if *keyPath == "" || *inPath == "" || *outPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sign -key PRIVATE_KEY -in FILE -out SIGNATURE")
		os.Exit(2)
	}

	keyData, err := os.ReadFile(*keyPath)
	if err != nil {
		fatal(err)
	}
	key, err := parsePrivateKey(keyData)
	if err != nil {
		fatal(err)
	}
	data, err := os.ReadFile(*inPath)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*outPath, ed25519.Sign(key, data), 0o644); err != nil {
		fatal(err)
	}
}

func parsePrivateKey(data []byte) (ed25519.PrivateKey, error) {
	if len(data) == ed25519.PrivateKeySize {
		return ed25519.PrivateKey(data), nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("private key must be a raw 64-byte Ed25519 key or PKCS#8 PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
	}
	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is %T, want Ed25519", key)
	}
	return edKey, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sign:", err)
	os.Exit(1)
}
