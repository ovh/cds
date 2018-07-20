package clair

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
)

func Delete(image reference.NamedTagged, manifest distribution.Manifest) error {
	layers, err := newLayering(image)
	if err != nil {
		return err
	}

	switch manifest.(type) {
	case schema1.SignedManifest:
		for _, l := range manifest.(schema1.SignedManifest).FSLayers {
			layers.digests = append(layers.digests, l.BlobSum.String())
		}
		return layers.deleteAll()
	case *schema1.SignedManifest:
		for _, l := range manifest.(*schema1.SignedManifest).FSLayers {
			layers.digests = append(layers.digests, l.BlobSum.String())
		}
		return layers.deleteAll()
	case schema2.DeserializedManifest:
		for _, l := range manifest.(schema2.DeserializedManifest).Layers {
			layers.digests = append(layers.digests, l.Digest.String())
		}
		return layers.deleteAll()
	case *schema2.DeserializedManifest:
		for _, l := range manifest.(*schema2.DeserializedManifest).Layers {
			layers.digests = append(layers.digests, l.Digest.String())
		}
		return layers.deleteAll()
	default:
		return errors.New("Unsupported Schema version.")
	}
}

func deleteLayer(id string) error {

	lURI := fmt.Sprintf("%v/layers/%v", uri, id)
	request, err := http.NewRequest("DELETE", lURI, nil)

	if err != nil {
		return fmt.Errorf("creating 'delete layer' request: %v", err)
	}
	response, err := (&http.Client{}).Do(request)

	if err != nil {
		return fmt.Errorf("delete layer %v: %v", id, err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("receiving http error: %d", response.StatusCode)
	}

	return nil
}
