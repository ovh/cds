package stash

import (
	"fmt"
	"testing"
)

func GetHook() string {
	return fmt.Sprintf(
		"%s?owner=%s&name=%s&branch=${refChange.name}&hash=${refChange.toHash}&message=${refChange.type}&author=",
		testURL,
		testProject,
		testRepo,
	)
}

func TestCreateHook(t *testing.T) {
	hook, err := client.Hooks.CreateHook(testProject, testRepo, hookKey, "GET", GetHook(), "", "", "")
	if err != nil {
		t.Errorf("Unexpected error on `client.Hooks.CreateHook()`, got %v", err)
	}
	if hook.Enabled != true {
		t.Error("Expected hook to be enabled")
	}
}

func TestCreateHookInvalidLink(t *testing.T) {
	hook, err := client.Hooks.CreateHook(testProject, testRepo, hookKey, "GET", "link", "", "", "")
	if err == nil {
		t.Error("Expected error on `client.Hooks.CreateHook()`")
	}
	if hook != nil {
		t.Errorf("Did not expect hook data, got %v", hook)
	}
}

func TestCreateHookInvalidProject(t *testing.T) {
	hook, err := client.Hooks.CreateHook("wrong", testRepo, hookKey, "GET", GetHook(), "", "", "")
	if err == nil {
		t.Error("Expected error on `client.Hooks.CreateHook()`")
	}
	if hook != nil {
		t.Errorf("Did not expect hook data, got %v", hook)
	}
}

func TestDeleteHook(t *testing.T) {
	err := client.Hooks.DeleteHook(testProject, testRepo, hookKey, GetHook())
	if err != nil {
		t.Errorf("Unexpected error on `client.Hooks.DeleteHook()`, got %v", err)
	}
}

func TestDeleteHookInvalidProject(t *testing.T) {
	err := client.Hooks.DeleteHook("wrong", testRepo, hookKey, GetHook())
	if err == nil {
		t.Error("Expected error on `client.Hooks.DeleteHook()`")
	}
}
