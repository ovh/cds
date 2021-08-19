package sdk_test

import (
	json "encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentUnmarshal(t *testing.T) {
	now := time.Unix(time.Now().Unix(), 0)

	nowBytes, err := json.Marshal(now)
	require.NoError(t, err)

	data1 := []byte(fmt.Sprintf("{\"name\":\"one\",\"last_modified\":%s}", nowBytes))
	data2 := []byte(fmt.Sprintf("{\"name\":\"two\",\"last_modified\":%d}", now.Unix()))

	var one sdk.Environment
	require.NoError(t, sdk.JSONUnmarshal(data1, &one))
	assert.Equal(t, "one", one.Name)
	assert.True(t, now.Equal(one.LastModified))

	var two sdk.Environment
	require.NoError(t, sdk.JSONUnmarshal(data2, &two))
	assert.Equal(t, "two", two.Name)
	assert.True(t, now.Equal(two.LastModified))
}
