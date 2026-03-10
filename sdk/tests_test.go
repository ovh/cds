package sdk

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJUnitTestsSuites_Trim_NoTrimNeeded(t *testing.T) {
	s := JUnitTestsSuites{
		TestSuites: []JUnitTestSuite{
			{
				Name:  "small suite",
				Total: 1,
				TestCases: []JUnitTestCase{
					{Name: "test1", Systemout: JUnitInnerResult{Value: "ok"}},
				},
			},
		},
	}
	trimmed := s.Trim()
	assert.Equal(t, "ok", trimmed.TestSuites[0].TestCases[0].Systemout.Value)
}

func TestJUnitTestsSuites_Trim_LargeData(t *testing.T) {
	// Create a test suite with > 1MB of systemout data
	largeLog := strings.Repeat("x", 2*1024*1024) // 2MB
	s := JUnitTestsSuites{
		TestSuites: []JUnitTestSuite{
			{
				Name:  "large suite",
				Total: 1,
				TestCases: []JUnitTestCase{
					{Name: "test1", Systemout: JUnitInnerResult{Value: largeLog}},
				},
			},
		},
	}

	trimmed := s.Trim()

	// Should be truncated, keeping the end (most recent logs)
	assert.True(t, len(trimmed.TestSuites[0].TestCases[0].Systemout.Value) < len(largeLog))
	assert.True(t, strings.HasPrefix(trimmed.TestSuites[0].TestCases[0].Systemout.Value, "[truncated]"))

	// Serialized size should be <= 1MB + some overhead for the suffix
	bts, _ := json.Marshal(trimmed)
	assert.LessOrEqual(t, len(bts), jUnitTestsSuitesMaxSizeBytes+1024) // small margin for truncated suffixes
}

func TestJUnitTestsSuites_Trim_MultipleFields(t *testing.T) {
	// Spread 3MB across multiple fields
	chunk := strings.Repeat("a", 512*1024) // 512KB each
	s := JUnitTestsSuites{
		TestSuites: []JUnitTestSuite{
			{
				Name:  "suite1",
				Total: 2,
				TestCases: []JUnitTestCase{
					{
						Name:      "test1",
						Systemout: JUnitInnerResult{Value: chunk},
						Systemerr: JUnitInnerResult{Value: chunk},
					},
					{
						Name: "test2",
						Failures: []JUnitTestFailure{
							{Message: "fail", Value: chunk},
						},
						Errors: []JUnitTestFailure{
							{Message: "err", Value: chunk},
						},
						Skipped: []JUnitTestSkipped{
							{Message: "skip", Value: chunk},
						},
					},
				},
			},
		},
	}

	trimmed := s.Trim()

	// All fields should be trimmed
	tc0 := trimmed.TestSuites[0].TestCases[0]
	tc1 := trimmed.TestSuites[0].TestCases[1]
	assert.True(t, strings.HasPrefix(tc0.Systemout.Value, "[truncated]"))
	assert.True(t, strings.HasPrefix(tc0.Systemerr.Value, "[truncated]"))
	assert.True(t, strings.HasPrefix(tc1.Failures[0].Value, "[truncated]"))
	assert.True(t, strings.HasPrefix(tc1.Errors[0].Value, "[truncated]"))
	assert.True(t, strings.HasPrefix(tc1.Skipped[0].Value, "[truncated]"))

	// Stats (counts) should be preserved
	assert.Equal(t, 2, trimmed.TestSuites[0].Total)
	assert.Equal(t, "suite1", trimmed.TestSuites[0].Name)
}

func TestGetDetailLightForContext_TestResult(t *testing.T) {
	largeLog := strings.Repeat("x", 100)
	testDetail := &V2WorkflowRunResultTestDetail{
		Name:   "results.xml",
		Size:   1000,
		MD5:    "abc",
		SHA1:   "def",
		SHA256: "ghi",
		TestsSuites: JUnitTestsSuites{
			TestSuites: []JUnitTestSuite{
				{
					Name:  "suite",
					Total: 1,
					TestCases: []JUnitTestCase{
						{Name: "test1", Systemout: JUnitInnerResult{Value: largeLog}},
					},
				},
			},
		},
		TestStats: TestsStats{Total: 1, TotalOK: 1},
	}

	result := V2WorkflowRunResult{
		Type: V2WorkflowRunResultTypeTest,
		Detail: V2WorkflowRunResultDetail{
			Data: testDetail,
			Type: "V2WorkflowRunResultTestDetail",
		},
	}

	detail, err := result.GetDetailLightForContext()
	require.NoError(t, err)

	lightDetail, ok := detail.(*V2WorkflowRunResultTestDetail)
	require.True(t, ok)

	// TestsSuites should be empty
	assert.Empty(t, lightDetail.TestsSuites.TestSuites)

	// Metadata should be preserved
	assert.Equal(t, "results.xml", lightDetail.Name)
	assert.Equal(t, int64(1000), lightDetail.Size)
	assert.Equal(t, "abc", lightDetail.MD5)
	assert.Equal(t, 1, lightDetail.TestStats.Total)

	// Original should be untouched
	assert.Len(t, testDetail.TestsSuites.TestSuites, 1)
}

func TestGetDetailLightForContext_NonTestResult(t *testing.T) {
	dockerDetail := &V2WorkflowRunResultDockerDetail{
		Name:      "myimage",
		HumanSize: "100MB",
	}

	result := V2WorkflowRunResult{
		Type: V2WorkflowRunResultTypeDocker,
		Detail: V2WorkflowRunResultDetail{
			Data: dockerDetail,
			Type: "V2WorkflowRunResultDockerDetail",
		},
	}

	detail, err := result.GetDetailLightForContext()
	require.NoError(t, err)

	// Should return the same detail unchanged
	d, ok := detail.(*V2WorkflowRunResultDockerDetail)
	require.True(t, ok)
	assert.Equal(t, "myimage", d.Name)
}
