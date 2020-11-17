package sdk

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEqualHook(t *testing.T) {
	h1 := NodeHook{
		UUID:          UUID(),
		HookModelName: RepositoryWebHookModelName,
		Config: WorkflowNodeHookConfig{
			HookConfigEventFilter: {
				Type:         HookConfigTypeMultiChoice,
				Configurable: true,
				Value:        "pr:opened",
			},
		},
	}
	h2 := NodeHook{
		UUID:          h1.UUID,
		HookModelName: RepositoryWebHookModelName,
		Config: WorkflowNodeHookConfig{
			HookConfigEventFilter: {
				Type:         HookConfigTypeMultiChoice,
				Configurable: true,
				Value:        "pr:merged",
			},
		},
	}
	require.False(t, h1.Equals(h2))
}
func TestNodeHook_ConfigValueContainsEventsDefault(t *testing.T) {
	var tests = []struct {
		given    []string
		expected bool
	}{
		{
			given:    BitbucketCloudEventsDefault,
			expected: true,
		},
		{
			given:    BitbucketCloudEventsDefault,
			expected: true,
		},
		{
			given:    GitHubEventsDefault,
			expected: true,
		},
		{
			given:    GitlabEventsDefault,
			expected: true,
		},
		{
			given:    GerritEventsDefault,
			expected: true,
		},
		{
			given:    GitHubEvents,
			expected: true,
		},
		{
			given:    []string{"foo", "bar"},
			expected: false,
		},
		{
			given:    []string{"push", "bar"}, // push is the default events for github
			expected: true,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test #%d with %+v", i, tt.given), func(t *testing.T) {
			h := &NodeHook{
				Config: WorkflowNodeHookConfig{
					HookConfigEventFilter: WorkflowNodeHookConfigValue{
						Value: strings.Join(tt.given, ";"),
					},
				},
			}
			require.Equal(t, tt.expected, h.ConfigValueContainsEventsDefault())
		})
	}
}
