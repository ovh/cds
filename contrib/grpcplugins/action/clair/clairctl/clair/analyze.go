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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/quay/clair/v2/api/v1"
)

var _ = prometheus.Handler()

//Analyze return Clair Image analysis
func Analyze(image reference.NamedTagged, manifest distribution.Manifest) (ImageAnalysis, error) {
	layers, err := newLayering(image)
	if err != nil {
		return ImageAnalysis{}, fmt.Errorf("ERROR cannot parse manifest")
	}

	switch manifest.(type) {
	case schema1.SignedManifest:
		for _, l := range manifest.(schema1.SignedManifest).FSLayers {
			layers.digests = append(layers.digests, l.BlobSum.String())
		}
		return layers.analyzeAll()
	case *schema1.SignedManifest:
		for _, l := range manifest.(*schema1.SignedManifest).FSLayers {
			layers.digests = append(layers.digests, l.BlobSum.String())
		}
		return layers.analyzeAll()
	case schema2.DeserializedManifest:
		for _, l := range manifest.(schema2.DeserializedManifest).Layers {
			layers.digests = append(layers.digests, l.Digest.String())
		}
		return layers.analyzeAll()
	case *schema2.DeserializedManifest:
		for _, l := range manifest.(*schema2.DeserializedManifest).Layers {
			layers.digests = append(layers.digests, l.Digest.String())
		}
		return layers.analyzeAll()
	default:
		return ImageAnalysis{}, fmt.Errorf("unsupported Schema version")
	}
}

func analyzeLayer(id string) (v1.LayerEnvelope, error) {
	lURI := fmt.Sprintf("%v/layers/%v?vulnerabilities", uri, id)
	response, err := http.Get(lURI)
	if err != nil {
		return v1.LayerEnvelope{}, fmt.Errorf("analysing layer %v: %v", id, err)
	}
	defer response.Body.Close()

	var analysis v1.LayerEnvelope
	err = json.NewDecoder(response.Body).Decode(&analysis)
	if err != nil {
		return v1.LayerEnvelope{}, fmt.Errorf("reading layer analysis: %v", err)
	}
	if response.StatusCode != 200 {
		//TODO(jgsqware): should I show response body in case of error?
		return v1.LayerEnvelope{}, fmt.Errorf("receiving http error: %d", response.StatusCode)
	}

	return analysis, nil
}
