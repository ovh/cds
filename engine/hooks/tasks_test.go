package hooks

import (
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_doWebHookExecutionStash(t *testing.T) {
	s := Service{}
	task := &TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &WebHookExecution{
			RequestBody: nil,
			RequestURL:  "uid=42413e87905b813a375c7043ce9d4047b7e265ae3730b60180cad02ae81cc62385e5b05b9e7c758b15bb3872498a5e88963f3deac308f636baf345ed9cf1b259&project=IRTM&name=rtm-packaging&branch=master&hash=123456789&message=monmessage&author=sguiheux",
		},
	}
	h, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	assert.Equal(t, "master", h.Payload["git.branch"])
	assert.Equal(t, "sguiheux", h.Payload["git.author"])
	assert.Equal(t, "monmessage", h.Payload["git.message"])
	assert.Equal(t, "123456789", h.Payload["git.hash"])
}
func Test_doWebHookExecutionGithub(t *testing.T) {
	s := Service{}
	task := &TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &WebHookExecution{
			RequestBody: []byte(githubPushEvent),
			RequestHeader: map[string][]string{
				"X-GitHub-Event": {"push"},
			},
			RequestURL: "",
		},
	}
	h, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	assert.Equal(t, "my-branch", h.Payload["git.branch"])
	assert.Equal(t, "baxterthehacker", h.Payload["git.author"])
	assert.Equal(t, "Update README.md", h.Payload["git.message"])
	assert.Equal(t, "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c", h.Payload["git.hash"])
	assert.Equal(t, "1", h.Payload["git.nb.commits"])
}

func Test_doWebHookExecutionGitlab(t *testing.T) {
	s := Service{}
	task := &TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &WebHookExecution{
			RequestBody: []byte(gitlabPushEvent),
			RequestHeader: map[string][]string{
				"X-Gitlab-Event": {"Push Hook"},
			},
			RequestURL: "",
		},
	}
	h, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	assert.Equal(t, "master", h.Payload["git.branch"])
	assert.Equal(t, "jsmith", h.Payload["git.author"])
	assert.Equal(t, "Update Catalan translation to e38cb41.", h.Payload["git.message"])
	assert.Equal(t, "da1560886d4f094c3e6c9ef40349f7d38b5d27d7", h.Payload["git.hash"])
	assert.Equal(t, "2", h.Payload["git.nb.commits"])
}

func Test_doWebHookExecutionBitbucker(t *testing.T) {
	s := Service{}
	task := &TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &WebHookExecution{
			RequestBody: []byte(bitbucketPushEvent),
			RequestHeader: map[string][]string{
				"X-Event-Key": {"repo:push"},
			},
			RequestURL: "",
		},
	}
	h, err := s.doWebHookExecution(task)
	test.NoError(t, err)

	assert.Equal(t, "name-of-branch", h.Payload["git.branch"])
	assert.Equal(t, "emmap1", h.Payload["git.author"])
	assert.Equal(t, "commit message\n", h.Payload["git.message"])
	assert.Equal(t, "709d658dc5b6d6afcd46049c2f332ee3f515a67d", h.Payload["git.hash"])
	assert.Equal(t, "1", h.Payload["git.nb.commits"])
}

var bitbucketPushEvent = `
	{
	"actor": {
		"type": "user",
		"username": "emmap1",
		"display_name": "Emma",
		"uuid": "{a54f16da-24e9-4d7f-a3a7-b1ba2cd98aa3}",
		"links": {
			"self": {
				"href": "https://api.bitbucket.org/api/2.0/users/emmap1"
			},
			"html": {
				"href": "https://api.bitbucket.org/emmap1"
			},
			"avatar": {
				"href": "https://bitbucket-api-assetroot.s3.amazonaws.com/c/photos/2015/Feb/26/3613917261-0-emmap1-avatar_avatar.png"
			}
		}
	},
	"repository": {
		"type": "repository",
		"links": {
			"self": {
				"href": "https://api.bitbucket.org/api/2.0/repositories/bitbucket/bitbucket"
			},
			"html": {
				"href": "https://api.bitbucket.org/bitbucket/bitbucket"
			},
			"avatar": {
				"href": "https://api-staging-assetroot.s3.amazonaws.com/c/photos/2014/Aug/01/bitbucket-logo-2629490769-3_avatar.png"
			}
		},
		"uuid": "{673a6070-3421-46c9-9d48-90745f7bfe8e}",
		"project": {},
		"full_name": "team_name/repo_name",
		"name": "repo_name",
		"website": "https://mywebsite.com/",
		"owner": {
			"type": "user",
			"username": "emmap1",
			"display_name": "Emma",
			"uuid": "{a54f16da-24e9-4d7f-a3a7-b1ba2cd98aa3}",
			"links": {
				"self": {
					"href": "https://api.bitbucket.org/api/2.0/users/emmap1"
				},
				"html": {
					"href": "https://api.bitbucket.org/emmap1"
				},
				"avatar": {
					"href": "https://bitbucket-api-assetroot.s3.amazonaws.com/c/photos/2015/Feb/26/3613917261-0-emmap1-avatar_avatar.png"
				}
			}
		},
		"scm": "git",
		"is_private": true
	},
	"push": {
		"changes": [{
			"new": {
				"type": "branch",
				"name": "name-of-branch",
				"target": {
					"type": "commit",
					"hash": "709d658dc5b6d6afcd46049c2f332ee3f515a67d",
					"author": {},
					"message": "new commit message\n",
					"date": "2015-06-09T03:34:49+00:00",
					"parents": [{
						"type": "commit",
						"hash": "1e65c05c1d5171631d92438a13901ca7dae9618c",
						"links": {
							"self": {
								"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/commit/8cbbd65829c7ad834a97841e0defc965718036a0"
							},
							"html": {
								"href": "https://bitbucket.org/user_name/repo_name/commits/8cbbd65829c7ad834a97841e0defc965718036a0"
							}
						}
					}],
					"links": {
						"self": {
							"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/commit/c4b2b7914156a878aa7c9da452a09fb50c2091f2"
						},
						"html": {
							"href": "https://bitbucket.org/user_name/repo_name/commits/c4b2b7914156a878aa7c9da452a09fb50c2091f2"
						}
					}
				},
				"links": {
					"self": {
						"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/refs/branches/master"
					},
					"commits": {
						"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/commits/master"
					},
					"html": {
						"href": "https://bitbucket.org/user_name/repo_name/branch/master"
					}
				}
			},
			"old": {
				"type": "branch",
				"name": "name-of-branch",
				"target": {
					"type": "commit",
					"hash": "1e65c05c1d5171631d92438a13901ca7dae9618c",
					"author": {},
					"message": "old commit message\n",
					"date": "2015-06-08T21:34:56+00:00",
					"parents": [{
						"type": "commit",
						"hash": "e0d0c2041e09746be5ce4b55067d5a8e3098c843",
						"links": {
							"self": {
								"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/commit/9c4a3452da3bc4f37af5a6bb9c784246f44406f7"
							},
							"html": {
								"href": "https://bitbucket.org/user_name/repo_name/commits/9c4a3452da3bc4f37af5a6bb9c784246f44406f7"
							}
						}
					}],
					"links": {
						"self": {
							"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/commit/b99ea6dad8f416e57c5ca78c1ccef590600d841b"
						},
						"html": {
							"href": "https://bitbucket.org/user_name/repo_name/commits/b99ea6dad8f416e57c5ca78c1ccef590600d841b"
						}
					}
				},
				"links": {
					"self": {
						"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/refs/branches/master"
					},
					"commits": {
						"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/commits/master"
					},
					"html": {
						"href": "https://bitbucket.org/user_name/repo_name/branch/master"
					}
				}
			},
			"links": {
				"html": {
					"href": "https://bitbucket.org/user_name/repo_name/branches/compare/c4b2b7914156a878aa7c9da452a09fb50c2091f2..b99ea6dad8f416e57c5ca78c1ccef590600d841b"
				},
				"diff": {
					"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/diff/c4b2b7914156a878aa7c9da452a09fb50c2091f2..b99ea6dad8f416e57c5ca78c1ccef590600d841b"
				},
				"commits": {
					"href": "https://api.bitbucket.org/2.0/repositories/user_name/repo_name/commits?include=c4b2b7914156a878aa7c9da452a09fb50c2091f2&exclude=b99ea6dad8f416e57c5ca78c1ccef590600d841b"
				}
			},
			"created": false,
			"forced": false,
			"closed": false,
			"commits": [{
				"hash": "03f4a7270240708834de475bcf21532d6134777e",
				"type": "commit",
				"message": "commit message\n",
				"author": {},
				"links": {
					"self": {
						"href": "https://api.bitbucket.org/2.0/repositories/user/repo/commit/03f4a7270240708834de475bcf21532d6134777e"
					},
					"html": {
						"href": "https://bitbucket.org/user/repo/commits/03f4a7270240708834de475bcf21532d6134777e"
					}
				}
			}],
			"truncated": false
		}]
	}
}
`

var gitlabPushEvent = `
	{
  "object_kind": "push",
  "before": "95790bf891e76fee5e1747ab589903a6a1f80f22",
  "after": "da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
  "ref": "refs/heads/master",
  "checkout_sha": "da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
  "user_id": 4,
  "user_name": "John Smith",
  "user_username": "jsmith",
  "user_email": "john@example.com",
  "user_avatar": "https://s.gravatar.com/avatar/d4c74594d841139328695756648b6bd6?s=8://s.gravatar.com/avatar/d4c74594d841139328695756648b6bd6?s=80",
  "project_id": 15,
  "project":{
    "id": 15,
    "name":"Diaspora",
    "description":"",
    "web_url":"http://example.com/mike/diaspora",
    "avatar_url":null,
    "git_ssh_url":"git@example.com:mike/diaspora.git",
    "git_http_url":"http://example.com/mike/diaspora.git",
    "namespace":"Mike",
    "visibility_level":0,
    "path_with_namespace":"mike/diaspora",
    "default_branch":"master",
    "homepage":"http://example.com/mike/diaspora",
    "url":"git@example.com:mike/diaspora.git",
    "ssh_url":"git@example.com:mike/diaspora.git",
    "http_url":"http://example.com/mike/diaspora.git"
  },
  "repository":{
    "name": "Diaspora",
    "url": "git@example.com:mike/diaspora.git",
    "description": "",
    "homepage": "http://example.com/mike/diaspora",
    "git_http_url":"http://example.com/mike/diaspora.git",
    "git_ssh_url":"git@example.com:mike/diaspora.git",
    "visibility_level":0
  },
  "commits": [
    {
      "id": "b6568db1bc1dcd7f8b4d5a946b0b91f9dacd7327",
      "message": "Update Catalan translation to e38cb41.",
      "timestamp": "2011-12-12T14:27:31+02:00",
      "url": "http://example.com/mike/diaspora/commit/b6568db1bc1dcd7f8b4d5a946b0b91f9dacd7327",
      "author": {
        "name": "Jordi Mallach",
        "email": "jordi@softcatala.org"
      },
      "added": ["CHANGELOG"],
      "modified": ["app/controller/application.rb"],
      "removed": []
    },
    {
      "id": "da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
      "message": "fixed readme",
      "timestamp": "2012-01-03T23:36:29+02:00",
      "url": "http://example.com/mike/diaspora/commit/da1560886d4f094c3e6c9ef40349f7d38b5d27d7",
      "author": {
        "name": "GitLab dev user",
        "email": "gitlabdev@dv6700.(none)"
      },
      "added": ["CHANGELOG"],
      "modified": ["app/controller/application.rb"],
      "removed": []
    }
  ],
  "total_commits_count": 4
}
`

var githubPushEvent = `
	{
  "ref": "refs/heads/my-branch",
  "before": "9049f1265b7d61be4a8904a9a27120d2064dab3b",
  "after": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
  "created": false,
  "deleted": false,
  "forced": false,
  "base_ref": null,
  "compare": "https://github.com/baxterthehacker/public-repo/compare/9049f1265b7d...0d1a26e67d8f",
  "commits": [
    {
      "id": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
      "tree_id": "f9d2a07e9488b91af2641b26b9407fe22a451433",
      "distinct": true,
      "message": "Update README.md",
      "timestamp": "2015-05-05T19:40:15-04:00",
      "url": "https://github.com/baxterthehacker/public-repo/commit/0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
      "author": {
        "name": "baxterthehacker",
        "email": "baxterthehacker@users.noreply.github.com",
        "username": "baxterthehacker"
      },
      "committer": {
        "name": "baxterthehacker",
        "email": "baxterthehacker@users.noreply.github.com",
        "username": "baxterthehacker"
      },
      "added": [

      ],
      "removed": [

      ],
      "modified": [
        "README.md"
      ]
    }
  ],
  "head_commit": {
    "id": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
    "tree_id": "f9d2a07e9488b91af2641b26b9407fe22a451433",
    "distinct": true,
    "message": "Update README.md",
    "timestamp": "2015-05-05T19:40:15-04:00",
    "url": "https://github.com/baxterthehacker/public-repo/commit/0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
    "author": {
      "name": "baxterthehacker",
      "email": "baxterthehacker@users.noreply.github.com",
      "username": "baxterthehacker"
    },
    "committer": {
      "name": "baxterthehacker",
      "email": "baxterthehacker@users.noreply.github.com",
      "username": "baxterthehacker"
    },
    "added": [

    ],
    "removed": [

    ],
    "modified": [
      "README.md"
    ]
  },
  "repository": {
    "id": 35129377,
    "name": "public-repo",
    "full_name": "baxterthehacker/public-repo",
    "owner": {
      "name": "baxterthehacker",
      "email": "baxterthehacker@users.noreply.github.com"
    },
    "private": false,
    "html_url": "https://github.com/baxterthehacker/public-repo",
    "description": "",
    "fork": false,
    "url": "https://github.com/baxterthehacker/public-repo",
    "forks_url": "https://api.github.com/repos/baxterthehacker/public-repo/forks",
    "keys_url": "https://api.github.com/repos/baxterthehacker/public-repo/keys{/key_id}",
    "collaborators_url": "https://api.github.com/repos/baxterthehacker/public-repo/collaborators{/collaborator}",
    "teams_url": "https://api.github.com/repos/baxterthehacker/public-repo/teams",
    "hooks_url": "https://api.github.com/repos/baxterthehacker/public-repo/hooks",
    "issue_events_url": "https://api.github.com/repos/baxterthehacker/public-repo/issues/events{/number}",
    "events_url": "https://api.github.com/repos/baxterthehacker/public-repo/events",
    "assignees_url": "https://api.github.com/repos/baxterthehacker/public-repo/assignees{/user}",
    "branches_url": "https://api.github.com/repos/baxterthehacker/public-repo/branches{/branch}",
    "tags_url": "https://api.github.com/repos/baxterthehacker/public-repo/tags",
    "blobs_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/blobs{/sha}",
    "git_tags_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/tags{/sha}",
    "git_refs_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/refs{/sha}",
    "trees_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/trees{/sha}",
    "statuses_url": "https://api.github.com/repos/baxterthehacker/public-repo/statuses/{sha}",
    "languages_url": "https://api.github.com/repos/baxterthehacker/public-repo/languages",
    "stargazers_url": "https://api.github.com/repos/baxterthehacker/public-repo/stargazers",
    "contributors_url": "https://api.github.com/repos/baxterthehacker/public-repo/contributors",
    "subscribers_url": "https://api.github.com/repos/baxterthehacker/public-repo/subscribers",
    "subscription_url": "https://api.github.com/repos/baxterthehacker/public-repo/subscription",
    "commits_url": "https://api.github.com/repos/baxterthehacker/public-repo/commits{/sha}",
    "git_commits_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/commits{/sha}",
    "comments_url": "https://api.github.com/repos/baxterthehacker/public-repo/comments{/number}",
    "issue_comment_url": "https://api.github.com/repos/baxterthehacker/public-repo/issues/comments{/number}",
    "contents_url": "https://api.github.com/repos/baxterthehacker/public-repo/contents/{+path}",
    "compare_url": "https://api.github.com/repos/baxterthehacker/public-repo/compare/{base}...{head}",
    "merges_url": "https://api.github.com/repos/baxterthehacker/public-repo/merges",
    "archive_url": "https://api.github.com/repos/baxterthehacker/public-repo/{archive_format}{/ref}",
    "downloads_url": "https://api.github.com/repos/baxterthehacker/public-repo/downloads",
    "issues_url": "https://api.github.com/repos/baxterthehacker/public-repo/issues{/number}",
    "pulls_url": "https://api.github.com/repos/baxterthehacker/public-repo/pulls{/number}",
    "milestones_url": "https://api.github.com/repos/baxterthehacker/public-repo/milestones{/number}",
    "notifications_url": "https://api.github.com/repos/baxterthehacker/public-repo/notifications{?since,all,participating}",
    "labels_url": "https://api.github.com/repos/baxterthehacker/public-repo/labels{/name}",
    "releases_url": "https://api.github.com/repos/baxterthehacker/public-repo/releases{/id}",
    "created_at": 1430869212,
    "updated_at": "2015-05-05T23:40:12Z",
    "pushed_at": 1430869217,
    "git_url": "git://github.com/baxterthehacker/public-repo.git",
    "ssh_url": "git@github.com:baxterthehacker/public-repo.git",
    "clone_url": "https://github.com/baxterthehacker/public-repo.git",
    "svn_url": "https://github.com/baxterthehacker/public-repo",
    "homepage": null,
    "size": 0,
    "stargazers_count": 0,
    "watchers_count": 0,
    "language": null,
    "has_issues": true,
    "has_downloads": true,
    "has_wiki": true,
    "has_pages": true,
    "forks_count": 0,
    "mirror_url": null,
    "open_issues_count": 0,
    "forks": 0,
    "open_issues": 0,
    "watchers": 0,
    "default_branch": "master",
    "stargazers": 0,
    "master_branch": "master"
  },
  "pusher": {
    "name": "baxterthehacker",
    "email": "baxterthehacker@users.noreply.github.com"
  },
  "sender": {
    "login": "baxterthehacker",
    "id": 6752317,
    "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3",
    "gravatar_id": "",
    "url": "https://api.github.com/users/baxterthehacker",
    "html_url": "https://github.com/baxterthehacker",
    "followers_url": "https://api.github.com/users/baxterthehacker/followers",
    "following_url": "https://api.github.com/users/baxterthehacker/following{/other_user}",
    "gists_url": "https://api.github.com/users/baxterthehacker/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/baxterthehacker/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/baxterthehacker/subscriptions",
    "organizations_url": "https://api.github.com/users/baxterthehacker/orgs",
    "repos_url": "https://api.github.com/users/baxterthehacker/repos",
    "events_url": "https://api.github.com/users/baxterthehacker/events{/privacy}",
    "received_events_url": "https://api.github.com/users/baxterthehacker/received_events",
    "type": "User",
    "site_admin": false
  }
}
`
