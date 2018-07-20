package clair

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/coreos/clair/api/v1"
	"github.com/docker/distribution/reference"
	"github.com/jgsqware/clairctl/config"
	"github.com/jgsqware/clairctl/docker/dockerdist"
	"github.com/jgsqware/clairctl/xstrings"
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
		log.Infof("using %v as local url", layer.hURL)
	}
	return &layer, nil
}

func (layers *layering) pushAll() error {
	layerCount := len(layers.digests)

	if layerCount == 0 {
		log.Warning("there is no layer to push")
	}
	for index, digest := range layers.digests {

		if config.IsLocal {
			digest = strings.TrimPrefix(digest, "sha256:")
		}

		lUID := xstrings.Substr(digest, 0, 12)
		log.Infof("Pushing Layer %d/%d [%v]", index+1, layerCount, lUID)
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
			log.Infof("adding layer %d/%d [%v]: %v", index+1, layerCount, lUID, err)
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

func (layers *layering) analyzeAll() ImageAnalysis {
	layerCount := len(layers.digests)
	res := []v1.LayerEnvelope{}

	for index := range layers.digests {
		digest := layers.digests[layerCount-index-1]
		if config.IsLocal {
			digest = strings.TrimPrefix(digest, "sha256:")
		}
		lShort := xstrings.Substr(digest, 0, 12)

		if a, err := analyzeLayer(digest); err != nil {
			log.Errorf("analysing layer [%v] %d/%d: %v", lShort, index+1, layerCount, err)
		} else {
			log.Infof("analysing layer [%v] %d/%d", lShort, index+1, layerCount)
			res = append(res, a)
		}
	}
	return ImageAnalysis{
		Registry:  xstrings.TrimPrefixSuffix(reference.Domain(layers.image), "http://", "/v2"),
		ImageName: layers.image.Name(),
		Tag:       layers.image.Tag(),
		Layers:    res,
	}
}

func (layers *layering) deleteAll() error {
	layerCount := len(layers.digests)

	if layerCount == 0 {
		logrus.Warningln("there is no layer to push")
	}

	for i := range layers.digests {
		digest := layers.digests[layerCount-i-1]
		if config.IsLocal {
			digest = strings.TrimPrefix(digest, "sha256:")
		}
		lShort := xstrings.Substr(digest, 0, 12)

		if err := deleteLayer(digest); err != nil {
			logrus.Infof("deleting layer [%v] %d/%d: Not found or already processed", lShort, i+1, layerCount)
		} else {
			logrus.Infof("deleting layer [%v] %d/%d", lShort, i+1, layerCount)
		}
	}
	return nil
}
