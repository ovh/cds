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
package docker

import (
	"errors"
	"reflect"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/config"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/docker/dockercli"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/docker/dockerdist"
)

//RetrieveManifest get manifest from local or remote docker registry
func RetrieveManifest(imageName string, withExport bool) (image reference.NamedTagged, manifest distribution.Manifest, err error) {
	if !config.IsLocal {
		image, manifest, err = dockerdist.DownloadManifest(imageName, true)
	} else {
		image, manifest, err = dockercli.GetLocalManifest(imageName, withExport)
	}
	return
}

//GetLayerDigests return layer digests from manifest schema1 and schema2
func GetLayerDigests(manifest distribution.Manifest) ([]digest.Digest, error) {
	layers := []digest.Digest{}

	switch manifest.(type) {
	case schema1.SignedManifest:
		for _, l := range manifest.(schema1.SignedManifest).FSLayers {
			layers = append(layers, l.BlobSum)
		}
	case *schema1.SignedManifest:
		for _, l := range manifest.(*schema1.SignedManifest).FSLayers {
			layers = append(layers, l.BlobSum)
		}
	case *schema2.DeserializedManifest:
		for _, d := range manifest.(*schema2.DeserializedManifest).Layers {
			layers = append(layers, d.Digest)
		}
	case schema2.DeserializedManifest:
		for _, d := range manifest.(schema2.DeserializedManifest).Layers {
			layers = append(layers, d.Digest)
		}
	default:
		return nil, errors.New("Not supported manifest schema type: " + reflect.TypeOf(manifest).String())
	}

	return layers, nil
}
