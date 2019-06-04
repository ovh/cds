// Copyright (C) 2017 Space Monkey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpsig

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

type Signer struct {
	id      string
	key     interface{}
	algo    Algorithm
	headers []string
}

// NewSigner contructs a signer with the specified key id, key, algorithm,
// and headers to sign. By default, if headers is nil or empty, the
// request-target and date headers will be signed.
func NewSigner(id string, key interface{}, algo Algorithm, headers []string) (
	signer *Signer) {

	s := &Signer{
		id:   id,
		key:  key,
		algo: algo,
	}

	// copy the headers slice, lowercasing as necessary
	if len(headers) == 0 {
		headers = []string{"(request-target)", "date"}
	}
	s.headers = make([]string, 0, len(headers))
	for _, header := range headers {
		s.headers = append(s.headers, strings.ToLower(header))
	}
	return s
}

// NewRSASHA1Signer contructs a signer with the specified key id, rsa private
// key and headers to sign.
func NewRSASHA1Signer(id string, key *rsa.PrivateKey, headers []string) (
	signer *Signer) {
	return NewSigner(id, key, RSASHA1, headers)
}

// NewRSASHA256Signer contructs a signer with the specified key id, rsa private
// key and headers to sign.
func NewRSASHA256Signer(id string, key *rsa.PrivateKey, headers []string) (
	signer *Signer) {
	return NewSigner(id, key, RSASHA256, headers)
}

// NewHMACSHA256Signer contructs a signer with the specified key id, hmac key,
// and headers to sign.
func NewHMACSHA256Signer(id string, key []byte, headers []string) (
	signer *Signer) {
	return NewSigner(id, key, HMACSHA256, headers)
}

// Sign signs an http request and adds the signature to the authorization header
func (r *Signer) Sign(req *http.Request) error {
	params, err := signRequest(r.id, r.key, r.algo, r.headers, req)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Signature "+params)
	return nil
}

// signRequest signs an http request and returns the parameter string.
func signRequest(id string, key interface{}, algo Algorithm, headers []string,
	req *http.Request) (params string, err error) {

	signature_data := BuildSignatureData(req, headers)

	signature, err := algo.Sign(key, signature_data)
	if err != nil {
		return "", err
	}

	// The headers parameter can be omitted if the only header is "Date". The
	// receiving end assumes ["date"] if no headers paramter is present.
	var headers_param string
	if !(len(headers) == 1 && headers[0] == "date") {
		headers_param = fmt.Sprintf("headers=%q,", strings.Join(headers, " "))
	}

	return fmt.Sprintf(
		"keyId=%q,algorithm=%q,%ssignature=%q",
		id,
		algo.Name(),
		headers_param,
		base64.StdEncoding.EncodeToString(signature)), nil
}
