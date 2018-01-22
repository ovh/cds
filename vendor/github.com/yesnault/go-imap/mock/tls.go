// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"sync"
	"time"
)

var tlsCfg = struct {
	sync.Mutex
	c, s *tls.Config
}{}

func clientTLS() *tls.Config {
	tlsCfg.Lock()
	defer tlsCfg.Unlock()
	if tlsCfg.c == nil {
		tlsCfg.c, tlsCfg.s = newConfig()
	}
	return tlsCfg.c
}

func serverTLS() *tls.Config {
	tlsCfg.Lock()
	defer tlsCfg.Unlock()
	if tlsCfg.s == nil {
		tlsCfg.c, tlsCfg.s = newConfig()
	}
	return tlsCfg.s
}

func newConfig() (client, server *tls.Config) {
	now := time.Now()
	tpl := x509.Certificate{
		SerialNumber:          new(big.Int).SetInt64(42),
		Subject:               pkix.Name{CommonName: ServerName},
		NotBefore:             now.Add(-2 * time.Hour).UTC(),
		NotAfter:              now.Add(2 * time.Hour).UTC(),
		BasicConstraintsValid: true,
		IsCA: true,
	}
	priv, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		panic(err)
	}
	crt, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &priv.PublicKey, priv)
	if err != nil {
		panic(err)
	}
	key := x509.MarshalPKCS1PrivateKey(priv)
	pair, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: crt}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: key}),
	)
	if err != nil {
		panic(err)
	}
	root, err := x509.ParseCertificate(crt)
	if err != nil {
		panic(err)
	}
	server = &tls.Config{Certificates: []tls.Certificate{pair}}
	client = &tls.Config{RootCAs: x509.NewCertPool(), ServerName: ServerName}
	client.RootCAs.AddCert(root)
	return
}
