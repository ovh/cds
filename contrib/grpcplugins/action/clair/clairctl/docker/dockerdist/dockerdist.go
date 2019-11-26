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

// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package dockerdist provides helper methods for retrieving and parsing a
// information from a remote Docker repository.
package dockerdist

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"

	"strings"

	distlib "github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/distribution"
	"github.com/docker/docker/registry"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

func isInsecureRegistry(registryHostname string) bool {
	for _, r := range viper.GetStringSlice("docker.insecure-registries") {
		if r == registryHostname {
			return true
		}
	}

	return false
}

func getService() (*registry.DefaultService, error) {
	serviceOptions := registry.ServiceOptions{
		InsecureRegistries: viper.GetStringSlice("docker.insecure-registries"),
	}
	return registry.NewService(serviceOptions)
}

// getRepositoryClient returns a client for performing registry operations against the given named
// image.
func getRepositoryClient(image reference.Named, insecure bool, scopes ...string) (distlib.Repository, error) {
	service, err := getService()
	if err != nil {
		fmt.Printf("Cannot get service: %v\n", err)
		return nil, err
	}

	ctx := context.Background()

	repoInfo, err := service.ResolveRepository(image)
	if err != nil {
		fmt.Printf("ResolveRepository err: %v\n", err)
		return nil, err
	}

	metaHeaders := map[string][]string{}
	endpoints, err := service.LookupPullEndpoints(reference.Domain(image))
	if err != nil {
		fmt.Printf("registry.LookupPullEndpoints error: %v\n", err)
		return nil, err
	}

	var confirmedV2 bool
	var repository distlib.Repository
	for _, endpoint := range endpoints {
		if confirmedV2 && endpoint.Version == registry.APIVersion1 {
			fmt.Printf("Skipping v1 endpoint %s because v2 registry was detected\n", endpoint.URL)
			continue
		}

		endpoint.TLSConfig.InsecureSkipVerify = viper.GetBool("auth.insecureSkipVerify")
		if isInsecureRegistry(endpoint.URL.Host) {
			endpoint.URL.Scheme = "http"
		}
		repository, confirmedV2, err = distribution.NewV2Repository(ctx, repoInfo, endpoint, metaHeaders, &types.AuthConfig{}, scopes...)
		if err != nil {
			fmt.Printf("cannot instantiate new v2 repository on %v\n", endpoint.URL)
			return nil, err
		}

		if !confirmedV2 {
			return nil, errors.New("Only V2 repository are supported")
		}
		break
	}

	return repository, nil
}

func GetPushURL(hostname string) (*url.URL, error) {
	service, err := getService()
	if err != nil {
		fmt.Printf("Cannot get service: %v\n", err)
		return nil, err
	}
	endpoints, err := service.LookupPushEndpoints(hostname)
	if err != nil {
		fmt.Printf("registry.LookupPushEndpoints error: %v\n", err)
		return nil, err
	}

	for _, endpoint := range endpoints {
		endpoint.TLSConfig.InsecureSkipVerify = viper.GetBool("auth.insecureSkipVerify")
		if isInsecureRegistry(endpoint.URL.Host) {
			endpoint.URL.Scheme = "http"
		}

		return url.Parse(endpoint.URL.String() + "/v2")
	}

	return nil, errors.New("No endpoints found")
}

// getDigest returns the digest for the given image.
func getDigest(ctx context.Context, repo distlib.Repository, image reference.Named) (digest.Digest, error) {
	if withDigest, ok := image.(reference.Canonical); ok {
		return withDigest.Digest(), nil
	}
	// Get TagService.
	tagSvc := repo.Tags(ctx)

	// Get Tag name.
	tag := "latest"
	if withTag, ok := image.(reference.NamedTagged); ok {
		tag = withTag.Tag()
	}

	// Get Tag's Descriptor.
	descriptor, err := tagSvc.Get(ctx, tag)
	if err != nil {

		// Docker returns an UnexpectedHTTPResponseError if it cannot parse the JSON body of an
		// unexpected error. Unfortunately, HEAD requests *by definition* don't have bodies, so
		// Docker will return this error for non-200 HEAD requests. We therefore have to hack
		// around it... *sigh*.
		if _, ok := err.(*client.UnexpectedHTTPResponseError); ok {
			return "", errors.New("Received error when trying to fetch the specified tag: it might not exist or you do not have access")
		}

		if strings.Contains(err.Error(), v2.ErrorCodeManifestUnknown.Message()) {
			return "", errors.New("this image or tag is not found")
		}

		return "", err
	}

	return descriptor.Digest, nil
}

// DownloadManifest the manifest for the given image, using the given credentials.
func DownloadManifest(image string, insecure bool) (reference.NamedTagged, distlib.Manifest, error) {
	fmt.Printf("Downloading manifest for %v\n", image)
	// Parse the image name as a docker image reference.
	n, err := reference.ParseNamed(image)
	if err != nil {
		return nil, nil, err
	}
	if reference.IsNameOnly(n) {
		n, _ = reference.ParseNamed(image + ":latest")
	}

	named := n.(reference.NamedTagged)
	// Create a reference to a repository client for the repo.
	repo, err := getRepositoryClient(named, insecure, "pull")
	if err != nil {
		return nil, nil, err
	}
	// Get the digest.
	ctx := context.Background()

	digest, err := getDigest(ctx, repo, named)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve the manifest for the tag.
	manSvc, err := repo.Manifests(ctx)
	if err != nil {
		return nil, nil, err
	}
	manifest, err := manSvc.Get(ctx, digest)
	if err != nil {
		return nil, nil, err
	}

	// Verify the manifest if it's signed.
	fmt.Printf("manifest type: %v\n", reflect.TypeOf(manifest))

	switch manifest.(type) {
	case *schema1.SignedManifest:
		_, verr := schema1.Verify(manifest.(*schema1.SignedManifest))
		if verr != nil {
			return nil, nil, verr
		}
	case *schema2.DeserializedManifest:
		fmt.Printf("retrieved schema2 manifest\n")
	default:
		log.Printf("Could not verify manifest for image %v: not signed", image)
	}

	return named, manifest, nil
}

// DownloadV1Manifest the manifest for the given image in v1 schema format, using the given credentials.
func DownloadV1Manifest(imageName string, insecure bool) (reference.NamedTagged, schema1.SignedManifest, error) {
	image, manifest, err := DownloadManifest(imageName, insecure)

	if err != nil {
		return nil, schema1.SignedManifest{}, err
	}
	// Ensure that the manifest type is supported.
	switch manifest.(type) {
	case *schema1.SignedManifest:
		return image, *manifest.(*schema1.SignedManifest), nil
	default:
		return nil, schema1.SignedManifest{}, errors.New("only v1 manifests are currently supported")
	}
}
