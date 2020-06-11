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
	"fmt"
	"strings"

	v1 "github.com/quay/clair/v2/api/v1"
	"github.com/docker/distribution/reference"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/config"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/docker/dockerdist"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/xstrings"
)

type layering struct {
	image          reference.NamedTagged
	digests        []string
	parentID, hURL string
}
 
func newLayering(image reference.NamedTagged) (*layering, error) {
	layer := layering{
		parentID: "",
		image:    image,
	}

	localIP, err := config.LocalServerIP()
	if err != nil {
		return nil, err
	}
	layer.hURL = fmt.Sprintf("http://%v/v2", localIP)
	if config.IsLocal {
		layer.hURL = strings.Replace(layer.hURL, "/v2", "/local", -1)
		fmt.Printf("using %v as local url\n", layer.hURL)
	}
	return &layer, nil
}

func (layers *layering) pushAll() error {
	layerCount := len(layers.digests)

	if layerCount == 0 {
		fmt.Printf("Warning: there is no layer to push\n")
	}
	for index, digest := range layers.digests {
		if config.IsLocal {
			digest = strings.TrimPrefix(digest, "sha256:")
		}

		lUID := xstrings.Substr(digest, 0, 12)
		fmt.Printf("Pushing Layer %d/%d [%v]\n", index+1, layerCount, lUID)
		domain := reference.Domain(layers.image)
		insertRegistryMapping(digest, domain)
		u, _ := dockerdist.GetPushURL(domain)
		payload := v1.LayerEnvelope{Layer: &v1.Layer{
			Name:       digest,
			Path:       blobsURI(u.String(), reference.Path(layers.image), digest),
			ParentName: layers.parentID,
			Format:     "Docker",
		}}

		//FIXME Update to TLS
		if config.IsLocal {
			local := layers.hURL + "/" + domain
			payload.Layer.Path = strings.Replace(payload.Layer.Path, u.String(), local, 1)
			payload.Layer.Path += "/layer.tar"
		}

		if err := pushLayer(payload); err != nil {
			fmt.Printf("adding layer %d/%d [%v]: %v\n", index+1, layerCount, lUID, err)
			if err != ErrUnanalizedLayer {
				return err
			}
			layers.parentID = ""
		} else {
			layers.parentID = payload.Layer.Name
		}
	}
	return nil
}

func (layers *layering) analyzeAll() (ImageAnalysis, error) {
	layerCount := len(layers.digests)
	res := []v1.LayerEnvelope{}

	for index := range layers.digests {
		digest := layers.digests[layerCount-index-1]
		if config.IsLocal {
			digest = strings.TrimPrefix(digest, "sha256:")
		}
		lShort := xstrings.Substr(digest, 0, 12)

		if a, err := analyzeLayer(digest); err != nil {
			return ImageAnalysis{}, fmt.Errorf("analysing layer [%v] %d/%d: %v", lShort, index+1, layerCount, err)
		} else {
			fmt.Printf("analysing layer [%v] %d/%d\n", lShort, index+1, layerCount)
			res = append(res, a)
		}
	}
	return ImageAnalysis{
		Registry:  xstrings.TrimPrefixSuffix(reference.Domain(layers.image), "http://", "/v2"),
		ImageName: layers.image.Name(),
		Tag:       layers.image.Tag(),
		Layers:    res,
	}, nil
}
