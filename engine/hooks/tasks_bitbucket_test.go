package hooks

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
)

func Test_doWebHookExecutionBitbucket(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPushEvent),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"repo:refs_changed"},
			},
			RequestURL: "",
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	assert.Equal(t, 1, len(hs))
	assert.Equal(t, "repo:refs_changed", hs[0].Payload["git.hook"])
	assert.Equal(t, "name-of-branch", hs[0].Payload["git.branch"])
	assert.Equal(t, "steven.guiheux", hs[0].Payload["git.author"])
	assert.Equal(t, "9f4fac7ec5642099982a86f584f2c4a362adb670", hs[0].Payload["git.hash"])
}

func Test_doWebHookExecutionBitbucketPRReviewerUpdated(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrReviewerUpdated),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:reviewer:updated"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved;pr:reviewer:updated",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:reviewer:updated", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])
}

func Test_doWebHookExecutionBitbucketPRReviewerApproved(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrReviewerApproved),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:reviewer:approved"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved;pr:reviewer:approved",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:reviewer:approved", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

	test.Equal(t, "francois.samin", hs[0].Payload[PR_REVIEWER])
	test.Equal(t, "francois.samin@foo", hs[0].Payload[PR_REVIEWER_EMAIL])
	test.Equal(t, "APPROVED", hs[0].Payload[PR_REVIEWER_STATUS])
	test.Equal(t, "REVIEWER", hs[0].Payload[PR_REVIEWER_ROLE])
	test.Equal(t, "UNAPPROVED", hs[0].Payload[PR_PREVIOUS_STATE])
}

func Test_doWebHookExecutionBitbucketPRReviewerUnapproved(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrReviewerUnapproved),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:reviewer:unapproved"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved;pr:reviewer:unapproved",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:reviewer:unapproved", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

	test.Equal(t, "francois.samin", hs[0].Payload[PR_REVIEWER])
	test.Equal(t, "francois.samin@foo", hs[0].Payload[PR_REVIEWER_EMAIL])
	test.Equal(t, "UNAPPROVED", hs[0].Payload[PR_REVIEWER_STATUS])
	test.Equal(t, "REVIEWER", hs[0].Payload[PR_REVIEWER_ROLE])
	test.Equal(t, "APPROVED", hs[0].Payload[PR_PREVIOUS_STATE])
}

func Test_doWebHookExecutionBitbucketPRReviewerNeedsWork(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrReviewerNeedsWorks),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:reviewer:needs_work"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved;pr:reviewer:needs_work",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:reviewer:needs_work", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

	test.Equal(t, "francois.samin", hs[0].Payload[PR_REVIEWER])
	test.Equal(t, "francois.samin@foo", hs[0].Payload[PR_REVIEWER_EMAIL])
	test.Equal(t, "NEEDS_WORK", hs[0].Payload[PR_REVIEWER_STATUS])
	test.Equal(t, "REVIEWER", hs[0].Payload[PR_REVIEWER_ROLE])
	test.Equal(t, "UNAPPROVED", hs[0].Payload[PR_PREVIOUS_STATE])
}

func Test_doWebHookExecutionBitbucketPRCommentAdded(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrCommentAdded),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:comment:added"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved;pr:comment:added",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:comment:added", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

	test.Equal(t, "my comment added", hs[0].Payload[PR_COMMENT_TEXT])
	test.Equal(t, "steven.guiheux", hs[0].Payload[PR_COMMENT_AUTHOR])
	test.Equal(t, "steven.guiheux@foo", hs[0].Payload[PR_COMMENT_AUTHOR_EMAIL])

}

func Test_doWebHookExecutionBitbucketPRCommentDeleted(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrCommentDeleted),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:comment:deleted"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved;pr:comment:deleted",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:comment:deleted", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

	test.Equal(t, "my comment deleted", hs[0].Payload[PR_COMMENT_TEXT])
	test.Equal(t, "steven.guiheux", hs[0].Payload[PR_COMMENT_AUTHOR])
	test.Equal(t, "steven.guiheux@foo", hs[0].Payload[PR_COMMENT_AUTHOR_EMAIL])

}

func Test_doWebHookExecutionBitbucketPRCommentModified(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrCommentModified),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:comment:edited"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved;pr:comment:edited",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:comment:edited", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

	test.Equal(t, "my comment edited", hs[0].Payload[PR_COMMENT_TEXT])
	test.Equal(t, "steven.guiheux", hs[0].Payload[PR_COMMENT_AUTHOR])
	test.Equal(t, "steven.guiheux@foo", hs[0].Payload[PR_COMMENT_AUTHOR_EMAIL])

	test.Equal(t, "moi aussi", hs[0].Payload[PR_COMMENT_TEXT_PREVIOUS])

}

func Test_doWebHookExecutionBitbucketPROpened(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrOpened),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:opened"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:approved",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:opened", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

}

func Test_doWebHookExecutionBitbucketPRModified(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrModified),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:modified"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:modified",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:modified", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "OPEN", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

	test.Equal(t, "Update README.md", hs[0].Payload[PR_PREVIOUS_TITLE])
	test.Equal(t, "prev_branch", hs[0].Payload[PR_PREVIOUS_BRANCH])
	test.Equal(t, "0987654321", hs[0].Payload[PR_PREVIOUS_HASH])
}

func Test_doWebHookExecutionBitbucketPRMerged(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrMerged),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:merged"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:opened;pr:merged",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:opened", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "MERGED", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])

}

func Test_doWebHookExecutionBitbucketPRDeleted(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrDeleted),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:deleted"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:deleted;pr:approved",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:deleted", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "DELETED", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])
}

func Test_doWebHookExecutionBitbucketPRDeclined(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketPrDeclined),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"pr:declined"},
			},
			RequestURL: "",
		},
		Config: map[string]sdk.WorkflowNodeHookConfigValue{
			sdk.HookConfigEventFilter: {
				Value: "pr:deleted;pr:declined",
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john doe", hs[0].Payload[GIT_AUTHOR])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[GIT_AUTHOR_EMAIL])

	test.Equal(t, "dest_branch", hs[0].Payload[GIT_BRANCH])
	test.Equal(t, "fork/repo", hs[0].Payload[GIT_REPOSITORY_BEFORE])
	test.Equal(t, "654321654321", hs[0].Payload[GIT_HASH])
	test.Equal(t, "6543216", hs[0].Payload[GIT_HASH_SHORT])

	test.Equal(t, "pr:declined", hs[0].Payload[GIT_EVENT])
	test.Equal(t, "src_branch", hs[0].Payload[GIT_BRANCH_BEFORE])
	test.Equal(t, "12345671234567", hs[0].Payload[GIT_HASH_BEFORE])
	test.Equal(t, "666", hs[0].Payload[PR_ID])
	test.Equal(t, "DECLINED", hs[0].Payload[PR_STATE])
	test.Equal(t, "My First PR", hs[0].Payload[PR_TITLE])
	test.Equal(t, "my/repo", hs[0].Payload[GIT_REPOSITORY])
}

func Test_doWebHookExecutionBitbucketMultiple(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketMultiplePushEvent),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"repo:refs_changed"},
			},
			RequestURL: "",
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	assert.Equal(t, 2, len(hs))
	assert.Equal(t, "name-of-branch", hs[0].Payload["git.branch"])
	assert.Equal(t, "steven.guiheux", hs[0].Payload["git.author"])
	assert.Equal(t, "9f4fac7ec5642099982a86f584f2c4a362adb670", hs[0].Payload["git.hash"])
	assert.Equal(t, "name-of-branch-bis", hs[1].Payload["git.branch"])
	assert.Equal(t, "steven.guiheux", hs[1].Payload["git.author"])
	assert.Equal(t, "9f4fac7ec5642099982a86f584f2c4a362adb670", hs[0].Payload["git.hash"])
}

var bitbucketPrReviewerUpdated = `
{
    "eventKey": "pr:reviewer:updated",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "addedReviewers": [
        {
            "name": "francois.samin",
            "emailAddress": "francois.samin@foo.bar",
            "id": 2427,
            "displayName": "François Samin",
            "active": true,
            "slug": "francois.samin",
            "type": "NORMAL"
        }
    ],
    "removedReviewers": []
}`
var bitbucketPrReviewerApproved = `
{
    "eventKey": "pr:reviewer:approved",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "participant": {
        "user": {
            "name": "francois.samin",
            "emailAddress": "francois.samin@foo",
            "id": 2427,
            "displayName": "François Samin",
            "active": true,
            "slug": "francois.samin",
            "type": "NORMAL"
        },
        "lastReviewedCommit": "80f3f7e9b9da3cb3d7f11145709323d8a65d2922",
        "role": "REVIEWER",
        "approved": false,
        "status": "APPROVED"
    },
    "previousStatus": "UNAPPROVED"
}`
var bitbucketPrReviewerUnapproved = `
{
    "eventKey": "pr:reviewer:unapproved",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "participant": {
        "user": {
            "name": "francois.samin",
            "emailAddress": "francois.samin@foo",
            "id": 2427,
            "displayName": "François Samin",
            "active": true,
            "slug": "francois.samin",
            "type": "NORMAL"
        },
        "lastReviewedCommit": "80f3f7e9b9da3cb3d7f11145709323d8a65d2922",
        "role": "REVIEWER",
        "approved": false,
        "status": "UNAPPROVED"
    },
    "previousStatus": "APPROVED"
}`
var bitbucketPrReviewerNeedsWorks = `
{
    "eventKey": "pr:reviewer:needs_work",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "participant": {
        "user": {
            "name": "francois.samin",
            "emailAddress": "francois.samin@foo",
            "id": 2427,
            "displayName": "François Samin",
            "active": true,
            "slug": "francois.samin",
            "type": "NORMAL"
        },
        "lastReviewedCommit": "80f3f7e9b9da3cb3d7f11145709323d8a65d2922",
        "role": "REVIEWER",
        "approved": false,
        "status": "NEEDS_WORK"
    },
    "previousStatus": "UNAPPROVED"
}`
var bitbucketPrCommentAdded = `
{
    "eventKey": "pr:comment:added",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "comment": {
        "properties": {
            "repositoryId": 7716
        },
        "id": 252969,
        "version": 1,
        "text": "my comment added",
        "author": {
            "name": "steven.guiheux",
            "emailAddress": "steven.guiheux@foo",
            "id": 1363,
            "displayName": "Steven Guiheux",
            "active": true,
            "slug": "steven.guiheux",
            "type": "NORMAL"
        },
        "createdDate": 1571042924878,
        "updatedDate": 1571044442062,
        "comments": [],
        "tasks": []
    }
}`
var bitbucketPrCommentDeleted = `
{
    "eventKey": "pr:comment:deleted",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "comment": {
        "properties": {
            "repositoryId": 7716
        },
        "id": 252969,
        "version": 1,
        "text": "my comment deleted",
        "author": {
            "name": "steven.guiheux",
            "emailAddress": "steven.guiheux@foo",
            "id": 1363,
            "displayName": "Steven Guiheux",
            "active": true,
            "slug": "steven.guiheux",
            "type": "NORMAL"
        },
        "createdDate": 1571042924878,
        "updatedDate": 1571044442062,
        "comments": [],
        "tasks": []
    }
}`
var bitbucketPrCommentModified = `
{
    "eventKey": "pr:comment:edited",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "comment": {
        "properties": {
            "repositoryId": 7716
        },
        "id": 252969,
        "version": 1,
        "text": "my comment edited",
        "author": {
            "name": "steven.guiheux",
            "emailAddress": "steven.guiheux@foo",
            "id": 1363,
            "displayName": "Steven Guiheux",
            "active": true,
            "slug": "steven.guiheux",
            "type": "NORMAL"
        },
        "createdDate": 1571042924878,
        "updatedDate": 1571044442062,
        "comments": [],
        "tasks": []
    },
    "previousComment": "moi aussi"
}`
var bitbucketPrModified = `
{
    "eventKey": "pr:modified",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    },
    "previousTitle": "Update README.md",
    "previousDescription": null,
    "previousTarget": {
        "id": "refs/heads/master",
        "displayId": "prev_branch",
        "type": "BRANCH",
        "latestCommit": "0987654321",
        "latestChangeset": "0987654321"
    }
}`
var bitbucketPrMerged = `
{
    "eventKey": "pr:opened",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "MERGED",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    }
}`
var bitbucketPrOpened = `
{
    "eventKey": "pr:opened",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "OPEN",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    }
}`
var bitbucketPrDeleted = `
{
    "eventKey": "pr:deleted",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "DELETED",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    }
}`
var bitbucketPushEvent = `
	{
    "eventKey": "repo:refs_changed",
    "date": "2017-11-30T15:24:01+0100",
    "actor": {
        "name": "steven.guiheux",
        "emailAddress": "steven.guiheux@foo.bar",
        "id": 1363,
        "displayName": "Steven Guiheux",
        "active": true,
        "slug": "steven.guiheux",
        "type": "NORMAL"
    },
    "repository": {
        "slug": "sseclient",
        "id": 6096,
        "name": "sseclient",
        "scmId": "git",
        "state": "AVAILABLE",
        "statusMessage": "Available",
        "forkable": true,
        "project": {
            "key": "~STEVEN.GUIHEUX",
            "id": 112,
            "name": "Steven Guiheux",
            "type": "PERSONAL",
            "owner": {
                "name": "steven.guiheux",
                "emailAddress": "steven.guiheux@foo.bar",
                "id": 1363,
                "displayName": "Steven Guiheux",
                "active": true,
                "slug": "steven.guiheux",
                "type": "NORMAL"
            }
        },
        "public": true
    },
    "changes": [
        {
            "ref": {
                "id": "refs/heads/name-of-branch",
                "displayId": "name-of-branch",
                "type": "BRANCH"
            },
            "refId": "refs/heads/name-of-branch",
            "fromHash": "0000000000000000000000000000000000000000",
            "toHash": "9f4fac7ec5642099982a86f584f2c4a362adb670",
            "type": "ADD"
        }
    ]
}
`
var bitbucketMultiplePushEvent = `
	{
    "eventKey": "repo:refs_changed",
    "date": "2017-11-30T15:24:01+0100",
    "actor": {
        "name": "steven.guiheux",
        "emailAddress": "steven.guiheux@foo.bar",
        "id": 1363,
        "displayName": "Steven Guiheux",
        "active": true,
        "slug": "steven.guiheux",
        "type": "NORMAL"
    },
    "repository": {
        "slug": "sseclient",
        "id": 6096,
        "name": "sseclient",
        "scmId": "git",
        "state": "AVAILABLE",
        "statusMessage": "Available",
        "forkable": true,
        "project": {
            "key": "~STEVEN.GUIHEUX",
            "id": 112,
            "name": "Steven Guiheux",
            "type": "PERSONAL",
            "owner": {
                "name": "steven.guiheux",
                "emailAddress": "steven.guiheux@foo.bar",
                "id": 1363,
                "displayName": "Steven Guiheux",
                "active": true,
                "slug": "steven.guiheux",
                "type": "NORMAL"
            }
        },
        "public": true
    },
    "changes": [
        {
            "ref": {
                "id": "refs/heads/name-of-branch",
                "displayId": "name-of-branch",
                "type": "BRANCH"
            },
            "refId": "refs/heads/name-of-branch",
            "fromHash": "0000000000000000000000000000000000000000",
            "toHash": "9f4fac7ec5642099982a86f584f2c4a362adb670",
            "type": "ADD"
        },
        {
            "ref": {
                "id": "refs/heads/name-of-branch-bis",
                "displayId": "name-of-branch-bis",
                "type": "BRANCH"
            },
            "refId": "refs/heads/name-of-branch-bis",
            "fromHash": "0000000000000000000000000000000000000000",
            "toHash": "9f4fac7ec5642099982a86f584f2c4a362adb670",
            "type": "ADD"
        }
    ]
}
`
var bitbucketPrDeclined = `
{
    "eventKey": "pr:declined",
    "date": "2019-10-15T09:33:38+0200",
    "actor": {
        "name": "john.doe",
        "emailAddress": "john.doe@targate.fr",
        "id": 1363,
        "displayName": "john doe",
        "active": true,
        "slug": "foo",
        "type": "NORMAL",
        "links": {
            "self": [
                {
                    "href": "https://bitbucket/users/bar"
                }
            ]
        }
    },
    "pullRequest": {
        "id": 666,
        "version": 0,
        "title": "My First PR",
        "state": "DECLINED",
        "open": true,
        "closed": false,
        "createdDate": 1569939813210,
        "updatedDate": 1569939813210,
        "fromRef": {
            "id": "refs/heads/workflowUpdate1",
            "displayId": "src_branch",
            "latestCommit": "12345671234567",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "FOO",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "fork",
                    "id": 112,
                    "name": "foo",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/foo/bar.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/foo/bar.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/foo/repos/bar/browse"
                        }
                    ]
                }
            }
        },
        "toRef": {
            "id": "refs/heads/dest_branch",
            "displayId": "dest_branch",
            "latestCommit": "654321654321",
            "repository": {
                "slug": "repo",
                "id": 11444,
                "name": "bar",
                "scmId": "git",
                "state": "AVAILABLE",
                "statusMessage": "Available",
                "forkable": true,
                "project": {
                    "key": "my",
                    "id": 112,
                    "name": "my",
                    "type": "PERSONAL",
                    "owner": {
                        "name": "foo",
                        "emailAddress": "foo@bar",
                        "id": 1363,
                        "displayName": "foo",
                        "active": true,
                        "slug": "foo",
                        "type": "NORMAL",
                        "links": {
                            "self": [
                                {
                                    "href": "https://bitbucket/users/bar"
                                }
                            ]
                        }
                    },
                    "links": {
                        "self": [
                            {
                                "href": "https://bitbucket/users/bar"
                            }
                        ]
                    }
                },
                "public": false,
                "links": {
                    "clone": [
                        {
                            "href": "https://bitbucket/scm/bar/foo.git",
                            "name": "http"
                        },
                        {
                            "href": "ssh://git@bitbucket:7999/bar/foo.git",
                            "name": "ssh"
                        }
                    ],
                    "self": [
                        {
                            "href": "https://bitbucket/users/steven.guiheux/repos/ascoderepo/browse"
                        }
                    ]
                }
            }
        },
        "locked": false,
        "author": {
            "user": {
                "name": "cds",
                "emailAddress": "foo@bar.fr",
                "id": 7898,
                "displayName": "cds",
                "active": true,
                "slug": "cds",
                "type": "NORMAL",
                "links": {
                    "self": [
                        {
                            "href": "http//foo/bar"
                        }
                    ]
                }
            },
            "role": "AUTHOR",
            "approved": false,
            "status": "UNAPPROVED"
        },
        "reviewers": [],
        "participants": [],
        "links": {
            "self": [
                {
                    "href": "fff"
                }
            ]
        }
    }
}`
