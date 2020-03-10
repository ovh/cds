package environment_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestImportInto_Variable(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	u := &sdk.AuthentifiedUser{
		Username: "foo",
	}

	proj := sdk.Project{
		Key:  "testimportenv",
		Name: "testimportenv",
	}

	project.Delete(db, cache, proj.Key)

	test.NoError(t, project.Insert(db, cache, &proj))

	env := sdk.Environment{
		Name:      "testenv",
		ProjectID: proj.ID,
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	v0 := sdk.Variable{
		Name:  "v0",
		Type:  sdk.StringVariable,
		Value: "value0",
	}

	v1 := sdk.Variable{
		Name:  "v1",
		Type:  sdk.StringVariable,
		Value: "value1",
	}

	v2 := sdk.Variable{
		Name:  "v2",
		Type:  sdk.StringVariable,
		Value: "value2",
	}

	test.NoError(t, environment.InsertVariable(db, env.ID, &v0, u))
	test.NoError(t, environment.InsertVariable(db, env.ID, &v1, u))
	test.NoError(t, environment.InsertVariable(db, env.ID, &v2, u))

	var err error
	env.Variables, err = environment.LoadAllVariables(db, env.ID)
	test.NoError(t, err)

	env2 := sdk.Environment{
		Name:      "testenv2",
		ProjectID: proj.ID,
		Variables: []sdk.Variable{
			{
				Name:  "v1",
				Type:  sdk.TextVariable,
				Value: "value1bis",
			},
			{
				Name:  "v2",
				Type:  sdk.StringVariable,
				Value: "value2bis",
			},
			{
				Name:  "v3",
				Type:  sdk.StringVariable,
				Value: "value3",
			},
		},
	}

	allMsg := []sdk.Message{}
	msgChan := make(chan sdk.Message)
	done := make(chan bool)

	go func() {
		for {
			msg, ok := <-msgChan
			allMsg = append(allMsg, msg)
			if !ok {
				done <- true
			}
		}
	}()

	environment.ImportInto(db, &env2, &env, msgChan, u)

	close(msgChan)
	<-done

	env3, err := environment.LoadEnvironmentByID(db, env.ID)
	assert.NoError(t, err)

	var v0found, v1found, v2found, v3found bool
	for _, v := range env3.Variables {
		if v.Name == "v0" {
			v0found = true
			assert.Equal(t, "value0", v.Value)
			assert.Equal(t, sdk.StringVariable, v.Type)
		}
		if v.Name == "v1" {
			v1found = true
			assert.Equal(t, "value1bis", v.Value)
			assert.Equal(t, sdk.TextVariable, v.Type)
		}
		if v.Name == "v2" {
			v2found = true
			assert.Equal(t, "value2bis", v.Value)
			assert.Equal(t, sdk.StringVariable, v.Type)
		}
		if v.Name == "v3" {
			v3found = true
			assert.Equal(t, "value3", v.Value)
			assert.Equal(t, sdk.StringVariable, v.Type)
		}
	}

	assert.True(t, v0found)
	assert.True(t, v1found)
	assert.True(t, v2found)
	assert.True(t, v3found)

}

func TestImportInto_Group(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	u := &sdk.AuthentifiedUser{
		Username: "foo",
	}

	proj := sdk.Project{
		Key:  "testimportenv",
		Name: "testimportenv",
	}

	project.Delete(db, cache, proj.Key)

	test.NoError(t, project.Insert(db, cache, &proj))

	oldEnv, _ := environment.LoadEnvironmentByName(db, proj.Key, "testenv")
	if oldEnv != nil {
		environment.DeleteEnvironment(db, oldEnv.ID)
	}

	env := sdk.Environment{
		Name:      "testenv",
		ProjectID: proj.ID,
	}

	test.NoError(t, environment.InsertEnvironment(db, &env))

	var err error
	env.Variables, err = environment.LoadAllVariables(db, env.ID)
	test.NoError(t, err)

	env2 := sdk.Environment{
		Name:      "testenv2",
		ProjectID: proj.ID,
	}

	allMsg := []sdk.Message{}
	msgChan := make(chan sdk.Message)
	done := make(chan bool)

	go func() {
		for {
			msg, ok := <-msgChan
			allMsg = append(allMsg, msg)
			if !ok {
				done <- true
			}
		}
	}()

	environment.ImportInto(db, &env2, &env, msgChan, u)

	close(msgChan)
	<-done

	_, err = environment.LoadEnvironmentByID(db, env.ID)
	assert.NoError(t, err)
}
