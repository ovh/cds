package hooks

import (
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
	hs, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	assert.Equal(t, 1, len(hs))
	assert.Equal(t, "name-of-branch", hs[0].Payload["git.branch"])
	assert.Equal(t, "steven.guiheux", hs[0].Payload["git.author"])
	assert.Equal(t, "9f4fac7ec5642099982a86f584f2c4a362adb670", hs[0].Payload["git.hash"])
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
	hs, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john.doe", hs[0].Payload[GIT_AUTHOR])
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
	hs, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john.doe", hs[0].Payload[GIT_AUTHOR])
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
	hs, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john.doe", hs[0].Payload[GIT_AUTHOR])
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
	hs, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	test.Equal(t, "john.doe", hs[0].Payload[CDS_TRIGGERED_BY_USERNAME])
	test.Equal(t, "john doe", hs[0].Payload[CDS_TRIGGERED_BY_FULLNAME])
	test.Equal(t, "john.doe@targate.fr", hs[0].Payload[CDS_TRIGGERED_BY_EMAIL])
	test.Equal(t, "john.doe", hs[0].Payload[GIT_AUTHOR])
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
	hs, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	assert.Equal(t, 2, len(hs))
	assert.Equal(t, "name-of-branch", hs[0].Payload["git.branch"])
	assert.Equal(t, "steven.guiheux", hs[0].Payload["git.author"])
	assert.Equal(t, "9f4fac7ec5642099982a86f584f2c4a362adb670", hs[0].Payload["git.hash"])
	assert.Equal(t, "name-of-branch-bis", hs[1].Payload["git.branch"])
	assert.Equal(t, "steven.guiheux", hs[1].Payload["git.author"])
	assert.Equal(t, "9f4fac7ec5642099982a86f584f2c4a362adb670", hs[0].Payload["git.hash"])
}

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
        "emailAddress": "steven.guiheux@corp.ovh.com",
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
                "emailAddress": "steven.guiheux@corp.ovh.com",
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
        "emailAddress": "steven.guiheux@corp.ovh.com",
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
                "emailAddress": "steven.guiheux@corp.ovh.com",
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
