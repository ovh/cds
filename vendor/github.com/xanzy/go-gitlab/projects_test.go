package gitlab

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestListProjects(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[{"id":1},{"id":2}]`)
	})

	opt := &ListProjectsOptions{
		ListOptions: ListOptions{2, 3},
		Archived:    Bool(true),
		OrderBy:     String("name"),
		Sort:        String("asc"),
		Search:      String("query"),
		Simple:      Bool(true),
		Visibility:  Visibility(PublicVisibility),
	}

	projects, _, err := client.Projects.ListProjects(opt)
	if err != nil {
		t.Errorf("Projects.ListProjects returned error: %v", err)
	}

	want := []*Project{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(want, projects) {
		t.Errorf("Projects.ListProjects returned %+v, want %+v", projects, want)
	}
}

func TestListOwnedProjects(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[{"id":1},{"id":2}]`)
	})

	opt := &ListProjectsOptions{
		ListOptions: ListOptions{2, 3},
		Archived:    Bool(true),
		OrderBy:     String("name"),
		Sort:        String("asc"),
		Search:      String("query"),
		Simple:      Bool(true),
		Owned:       Bool(true),
		Visibility:  Visibility(PublicVisibility),
	}

	projects, _, err := client.Projects.ListProjects(opt)
	if err != nil {
		t.Errorf("Projects.ListOwnedProjects returned error: %v", err)
	}

	want := []*Project{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(want, projects) {
		t.Errorf("Projects.ListOwnedProjects returned %+v, want %+v", projects, want)
	}
}

func TestListStarredProjects(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[{"id":1},{"id":2}]`)
	})

	opt := &ListProjectsOptions{
		ListOptions: ListOptions{2, 3},
		Archived:    Bool(true),
		OrderBy:     String("name"),
		Sort:        String("asc"),
		Search:      String("query"),
		Simple:      Bool(true),
		Starred:     Bool(true),
		Visibility:  Visibility(PublicVisibility),
	}

	projects, _, err := client.Projects.ListProjects(opt)
	if err != nil {
		t.Errorf("Projects.ListStarredProjects returned error: %v", err)
	}

	want := []*Project{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(want, projects) {
		t.Errorf("Projects.ListStarredProjects returned %+v, want %+v", projects, want)
	}
}

func TestGetProject_byID(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"id":1}`)
	})
	want := &Project{ID: 1}

	project, _, err := client.Projects.GetProject(1)
	if err != nil {
		t.Fatalf("Projects.GetProject returns an error: %v", err)
	}

	if !reflect.DeepEqual(want, project) {
		t.Errorf("Projects.GetProject returned %+v, want %+v", project, want)
	}
}

func TestGetProject_byName(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects/", func(w http.ResponseWriter, r *http.Request) {
		testURL(t, r, "/projects/namespace%2Fname")
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"id":1}`)
	})
	want := &Project{ID: 1}

	project, _, err := client.Projects.GetProject("namespace/name")
	if err != nil {
		t.Fatalf("Projects.GetProject returns an error: %v", err)
	}

	if !reflect.DeepEqual(want, project) {
		t.Errorf("Projects.GetProject returned %+v, want %+v", project, want)
	}
}

func TestCreateProject(t *testing.T) {
	mux, server, client := setup()
	defer teardown(server)

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprint(w, `{"id":1}`)
	})

	opt := &CreateProjectOptions{Name: String("n")}

	project, _, err := client.Projects.CreateProject(opt)
	if err != nil {
		t.Errorf("Projects.CreateProject returned error: %v", err)
	}

	want := &Project{ID: 1}
	if !reflect.DeepEqual(want, project) {
		t.Errorf("Projects.CreateProject returned %+v, want %+v", project, want)
	}
}
