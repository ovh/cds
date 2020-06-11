/*

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.

CODE FROM https://github.com/jgsqware/clairctl

*/
package clair

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	v1 "github.com/quay/clair/v2/api/v1"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/config"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/docker/dockerdist"
	"github.com/spf13/viper"
)

// ErrUnanalizedLayer is returned when the layer was not correctly analyzed
var ErrUnanalizedLayer = errors.New("layer cannot be analyzed")

var registryMapping map[string]string

func Push(image reference.NamedTagged, manifest distribution.Manifest) error {
	layers, err := newLayering(image)
	if err != nil {
		return err
	}

	switch manifest.(type) {
	case schema1.SignedManifest:
		for _, l := range manifest.(schema1.SignedManifest).FSLayers {
			layers.digests = append(layers.digests, l.BlobSum.String())
		}
		return layers.pushAll()
	case *schema1.SignedManifest:
		for _, l := range manifest.(*schema1.SignedManifest).FSLayers {
			layers.digests = append(layers.digests, l.BlobSum.String())
		}
		return layers.pushAll()
	case schema2.DeserializedManifest:
		for _, l := range manifest.(schema2.DeserializedManifest).Layers {
			layers.digests = append(layers.digests, l.Digest.String())
		}
		return layers.pushAll()
	case *schema2.DeserializedManifest:
		for _, l := range manifest.(*schema2.DeserializedManifest).Layers {
			layers.digests = append(layers.digests, l.Digest.String())
		}
		return layers.pushAll()
	default:
		return errors.New("unsupported Schema version")
	}
}

func pushLayer(layer v1.LayerEnvelope) error {
	if !config.IsLocal {
		var err error
		layer, err = auth(layer)
		if err != nil {
			return fmt.Errorf("pushLayer error auth: %v", err)
		}
	}

	lJSON, err := json.Marshal(layer)
	if err != nil {
		return fmt.Errorf("marshalling layer: %v", err)
	}

	lURI := fmt.Sprintf("%v/layers", uri)
	request, err := http.NewRequest("POST", lURI, bytes.NewBuffer(lJSON))
	if err != nil {
		return fmt.Errorf("creating 'add layer' request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := (&http.Client{}).Do(request)

	if err != nil {
		return fmt.Errorf("pushing layer to clair: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 201 {
		if response.StatusCode == 422 {
			return ErrUnanalizedLayer
		}
		return fmt.Errorf("receiving http error: %d", response.StatusCode)
	}

	return nil
}

func auth(layer v1.LayerEnvelope) (v1.LayerEnvelope, error) {
	out, _ := url.Parse(layer.Layer.Path)
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: viper.GetBool("auth.insecureSkipVerify")},
		DisableCompression: true,
	}}

	req, _ := http.NewRequest("HEAD", out.String(), nil)

	resp, err := client.Do(req)
	if err != nil {
		return v1.LayerEnvelope{}, fmt.Errorf("response error: %v", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		dockerdist.AuthenticateResponse(client, resp, req)
	}
	layer.Layer.Headers = make(map[string]string)
	layer.Layer.Headers["Authorization"] = req.Header.Get("Authorization")
	return layer, nil
}

func blobsURI(registry string, name string, digest string) string {
	return strings.Join([]string{registry, name, "blobs", digest}, "/")
}

func insertRegistryMapping(layerDigest string, registryURI string) {
	hostURL, _ := dockerdist.GetPushURL(registryURI)
	registryMapping[layerDigest] = hostURL.String()
}

func init() {
	registryMapping = map[string]string{}
}
