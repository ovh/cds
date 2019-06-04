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
	"fmt"
	"net/http"
	"strings"
)

func RequireSignature(h http.Handler, v *Verifier, realm string) (
	out http.Handler) {

	var challenge_params []string
	if realm != "" {
		challenge_params = append(challenge_params,
			fmt.Sprintf("realm=%q", realm))
	}
	if headers := v.RequiredHeaders(); len(headers) > 0 {
		challenge_params = append(challenge_params,
			fmt.Sprintf("headers=%q", strings.Join(headers, " ")))
	}

	challenge := "Signature"
	if len(challenge_params) > 0 {
		challenge += fmt.Sprintf(" %s", strings.Join(challenge_params, ", "))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := v.Verify(req)
		if err != nil {
			w.Header()["WWW-Authenticate"] = []string{challenge}
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, err.Error())
			return
		}
		h.ServeHTTP(w, req)
	})
}
