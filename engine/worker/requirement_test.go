package main

import (
	"os"
	"testing"

	"github.com/ovh/cds/sdk"
)

func TestCheckRequirement(t *testing.T) {
	r := sdk.Requirement{
		Name:  "Go",
		Type:  sdk.BinaryRequirement,
		Value: "go",
	}

	ok, err := checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}
	if !ok {
		t.Fatalf("Requirement go should be here")
	}

	r.Value = "foo"
	ok, err = checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}
	if ok {
		t.Fatalf("Requirement foo should not be ok")
	}
}

func TestCheckHostnameRequirement(t *testing.T) {
	h, err := os.Hostname()
	if err != nil {
		// Meh, no way to test it
		t.Skip()
		return
	}
	r := sdk.Requirement{
		Type:  sdk.HostnameRequirement,
		Value: h,
	}

	ok, err := checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}

	if !ok {
		t.Fatalf("Requirement should be ok")
	}

	r.Value = "fewfewf"
	ok, err = checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}

	if ok {
		t.Fatalf("Requirement should not be ok")
	}
}

func TestNetworkAccessRequirement(t *testing.T) {
	r := sdk.Requirement{
		Type:  sdk.NetworkAccessRequirement,
		Value: "google.com:443",
	}

	ok, err := checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}

	if !ok {
		t.Fatalf("Requirement should be ok")
	}

	r.Value = "fewfewf"
	ok, err = checkRequirement(nil, r)
	if err != nil {
		t.Fatalf("checkRequirement should not fail: %s", err)
	}

	if ok {
		t.Fatalf("Requirement should not be ok")
	}
}
