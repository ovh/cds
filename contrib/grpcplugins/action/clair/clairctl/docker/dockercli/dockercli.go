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
package dockercli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/client"
	"github.com/docker/docker/image"
	"github.com/docker/docker/layer"
	"github.com/mholt/archiver"
	"github.com/opencontainers/go-digest"

	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/config"
)

//GetLocalManifest retrieve manifest for local image
func GetLocalManifest(imageName string, withExport bool) (reference.NamedTagged, distribution.Manifest, error) {
	n, err := reference.ParseNamed(imageName)
	if err != nil {
		return nil, nil, err
	}
	var image reference.NamedTagged
	if reference.IsNameOnly(n) {
		r, err := reference.WithTag(n, "latest")
		if err != nil {
			return nil, nil, err
		}
		image = r.(reference.NamedTagged)
	} else {
		image = n.(reference.NamedTagged)
	}
	if err != nil {
		return nil, nil, err
	}
	var manifest distribution.Manifest
	if withExport {
		manifest, err = save(image.Name() + ":" + image.Tag())
	} else {
		manifest, err = historyFromCommand(image.Name() + ":" + image.Tag())
	}

	if err != nil {
		return nil, schema1.SignedManifest{}, err
	}
	m := manifest.(schema1.SignedManifest)
	m.Name = image.Name()
	m.Tag = image.Tag()
	return image, m, err
}

func save(imageName string) (distribution.Manifest, error) {
	path := config.TmpLocal() + "/" + strings.Split(imageName, ":")[0] + "/blobs"

	if _, err := os.Stat(path); os.IsExist(err) {
		err := os.RemoveAll(path)
		if err != nil {
			return nil, err
		}
	}

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	img, err := cli.ImageSave(context.Background(), []string{imageName})
	if err != nil {
		return nil, fmt.Errorf("cannot save image %s: %s", imageName, err)
	}
	all, err := ioutil.ReadAll(img)
	if err != nil {
		panic(err)
	}
	img.Close()

	fo, err := os.Create(path + "/output.tar")
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	if err != nil {
		return nil, err
	}

	if _, err := fo.Write(all); err != nil {
		panic(err)
	}

	err = openAndUntar(path+"/output.tar", path)
	if err != nil {
		return nil, err
	}

	err = os.Remove(path + "/output.tar")
	if err != nil {
		return nil, err
	}
	return historyFromManifest(path)
}

func historyFromManifest(path string) (distribution.Manifest, error) {
	mf, err := os.Open(path + "/manifest.json")
	defer mf.Close()

	if err != nil {
		return schema1.SignedManifest{}, err
	}

	// https://github.com/docker/docker/blob/master/image/tarexport/tarexport.go#L17
	type manifestItem struct {
		Config       string
		RepoTags     []string
		Layers       []string
		Parent       image.ID                                 `json:",omitempty"`
		LayerSources map[layer.DiffID]distribution.Descriptor `json:",omitempty"`
	}

	var manifest []manifestItem
	if err = json.NewDecoder(mf).Decode(&manifest); err != nil {
		return schema1.SignedManifest{}, err
	} else if len(manifest) != 1 {
		return schema1.SignedManifest{}, err
	}
	var layers []string
	for _, layer := range manifest[0].Layers {
		layers = append(layers, strings.TrimSuffix(layer, "/layer.tar"))
	}
	var m schema1.SignedManifest

	for _, layer := range manifest[0].Layers {
		var d digest.Digest
		d, err := digest.Parse("sha256:" + strings.TrimSuffix(layer, "/layer.tar"))
		if err != nil {
			return schema1.SignedManifest{}, err
		}
		m.FSLayers = append(m.FSLayers, schema1.FSLayer{BlobSum: d})
	}

	return m, nil
}

func historyFromCommand(imageName string) (schema1.SignedManifest, error) {
	client, err := client.NewEnvClient()
	if err != nil {
		return schema1.SignedManifest{}, err
	}
	histories, err := client.ImageHistory(context.Background(), imageName)
	if err != nil {
		return schema1.SignedManifest{}, err
	}

	manifest := schema1.SignedManifest{}
	for _, history := range histories {
		var d digest.Digest
		d, err := digest.Parse(history.ID)
		if err != nil {
			return schema1.SignedManifest{}, err
		}
		manifest.FSLayers = append(manifest.FSLayers, schema1.FSLayer{BlobSum: d})
	}
	return manifest, nil
}

func openAndUntar(name, dst string) error {
	return archiver.Unarchive(name, dst)
}
