package workflowtemplate

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	db := &test.SqlExecutorMock{}

	expected := "SELECT * FROM workflow_template WHERE false"

	_, err := Get(db, NewCriteria())
	assert.Nil(t, err)
	assert.Equal(t, expected, db.LastQuery().Query)

	_, err = GetAll(db, NewCriteria())
	assert.Nil(t, err)
	assert.Equal(t, expected, db.LastQuery().Query)
}

func TestAudit(t *testing.T) {
	db := &test.SqlExecutorMock{}

	expected := "SELECT * FROM workflow_template_audit WHERE false ORDER BY created ASC"

	_, err := GetAudits(db, NewCriteriaAudit())
	assert.Nil(t, err)
	assert.Equal(t, expected, db.LastQuery().Query)
}

func TestInstance(t *testing.T) {
	db := &test.SqlExecutorMock{}

	expected := "SELECT * FROM workflow_template_instance WHERE false"

	_, err := GetInstance(db, NewCriteriaInstance())
	assert.Nil(t, err)
	assert.Equal(t, expected, db.LastQuery().Query)

	_, err = GetInstances(db, NewCriteriaInstance())
	assert.Nil(t, err)
	assert.Equal(t, expected, db.LastQuery().Query)
}

func TestInstanceAudit(t *testing.T) {
	db := &test.SqlExecutorMock{}

	expected := "SELECT * FROM workflow_template_instance_audit WHERE false ORDER BY created ASC"

	_, err := GetInstanceAudits(db, NewCriteriaInstanceAudit())
	assert.Nil(t, err)
	assert.Equal(t, expected, db.LastQuery().Query)
}
