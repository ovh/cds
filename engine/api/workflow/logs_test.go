package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func Test_truncateLogs(t *testing.T) {
	logs := &sdk.Log{Val: "12345"}
	assert.Equal(t, false, truncateLogs(15, 0, logs))
	assert.Equal(t, "12345", logs.Val)

	logs = &sdk.Log{Val: "12345678901234567890"}
	assert.Equal(t, false, truncateLogs(15, 0, logs))
	assert.Equal(t, "123456789012345... truncated\n", logs.Val)

	logs = &sdk.Log{Val: "12345678901234567890"}
	assert.Equal(t, false, truncateLogs(15, 5, logs))
	assert.Equal(t, "1234567890... truncated\n", logs.Val)

	assert.Equal(t, true, truncateLogs(15, 20, logs))
}

func Test_truncateStepLogs(t *testing.T) {
	logs := &sdk.ServiceLog{Val: "12345"}
	assert.Equal(t, false, truncateServiceLogs(15, 0, logs))
	assert.Equal(t, "12345", logs.Val)

	logs = &sdk.ServiceLog{Val: "12345678901234567890"}
	assert.Equal(t, false, truncateServiceLogs(15, 0, logs))
	assert.Equal(t, "123456789012345... truncated\n", logs.Val)

	logs = &sdk.ServiceLog{Val: "12345678901234567890"}
	assert.Equal(t, false, truncateServiceLogs(15, 5, logs))
	assert.Equal(t, "1234567890... truncated\n", logs.Val)

	assert.Equal(t, true, truncateServiceLogs(15, 20, logs))
}
