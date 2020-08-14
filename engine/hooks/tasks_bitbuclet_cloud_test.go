package hooks

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
)

func Test_doWebHookExecutionBitbucketCloud(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeRepoManagerWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: []byte(bitbucketCloudPush),
			RequestHeader: map[string][]string{
				BitbucketHeader: {"repo:push"},
			},
			RequestURL: "",
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	assert.Equal(t, 1, len(hs))
	assert.Equal(t, "repo:push", hs[0].Payload[GIT_EVENT])
	assert.Equal(t, "foo", hs[0].Payload[GIT_BRANCH])
	assert.Equal(t, "Vv", hs[0].Payload[GIT_AUTHOR])
	assert.Equal(t, "77d120bd9980621d506240832dbbd7b3a28c5717", hs[0].Payload[GIT_HASH])
	assert.Equal(t, "repo1/testhook", hs[0].Payload[GIT_REPOSITORY])
}

var bitbucketCloudPush = `
{
  "push": {
    "changes": [
      {
        "forced": false,
        "old": null,
        "links": {
          "commits": {
            "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/commits?include=77d120bd9980621d506240832dbbd7b3a28c5717"
          },
          "html": {
            "href": "https://bitbucket.org/repo1/testhook/branch/foo"
          }
        },
        "created": true,
        "commits": [
          {
            "rendered": {},
            "hash": "77d120bd9980621d506240832dbbd7b3a28c5717",
            "links": {
              "self": {
                "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/commit/77d120bd9980621d506240832dbbd7b3a28c5717"
              },
              "comments": {
                "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/commit/77d120bd9980621d506240832dbbd7b3a28c5717/comments"
              },
              "patch": {
                "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/patch/77d120bd9980621d506240832dbbd7b3a28c5717"
              },
              "html": {
                "href": "https://bitbucket.org/repo1/testhook/commits/77d120bd9980621d506240832dbbd7b3a28c5717"
              },
              "diff": {
                "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/diff/77d120bd9980621d506240832dbbd7b3a28c5717"
              },
              "approve": {
                "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/commit/77d120bd9980621d506240832dbbd7b3a28c5717/approve"
              },
              "statuses": {
                "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/commit/77d120bd9980621d506240832dbbd7b3a28c5717/statuses"
              }
            },
            "author": {
              "raw": "Vv <foo.bar@rr.fr>",
              "type": "author",
              "user": {
                "display_name": "Vv",
                "account_id": "5b98adc8b2b15c2bdfcccf8a",
                "links": {
                  "self": {
                    "href": "https://api.bitbucket.org/2.0/users/%blabla%7D"
                  },
                  "html": {
                    "href": "https://bitbucket.org/%blabla%7D/"
                  },
                  "avatar": {
                    "href": "https://secure.gravatar.com/avatar/fdfsfsd?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FSG-6.png"
                  }
                },
                "nickname": "repo1",
                "type": "user",
                "uuid": "{6485bd54-5433-45c5-a671-fc35b4166a49}"
              }
            },
            "summary": {
              "raw": ".gitignore created online with Bitbucket",
              "markup": "markdown",
              "html": "<p>.gitignore created online with Bitbucket</p>",
              "type": "rendered"
            },
            "parents": [],
            "date": "2019-10-16T12:15:54+00:00",
            "message": ".gitignore created online with Bitbucket",
            "type": "commit",
            "properties": {}
          }
        ],
        "truncated": false,
        "closed": false,
        "new": {
          "name": "foo",
          "links": {
            "commits": {
              "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/commits/foo"
            },
            "self": {
              "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/refs/branches/foo"
            },
            "html": {
              "href": "https://bitbucket.org/repo1/testhook/branch/foo"
            }
          },
          "default_merge_strategy": "merge_commit",
          "merge_strategies": [
            "merge_commit",
            "squash",
            "fast_forward"
          ],
          "type": "branch",
          "target": {
            "rendered": {},
            "hash": "77d120bd9980621d506240832dbbd7b3a28c5717",
            "links": {
              "self": {
                "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook/commit/77d120bd9980621d506240832dbbd7b3a28c5717"
              },
              "html": {
                "href": "https://bitbucket.org/repo1/testhook/commits/77d120bd9980621d506240832dbbd7b3a28c5717"
              }
            },
            "author": {
              "raw": "Vv <foo.bar@rr.fr>",
              "type": "author",
              "user": {
                "display_name": "Vv",
                "account_id": "5b98adc8b2b15c2bdfcccf8a",
                "links": {
                  "self": {
                    "href": "https://api.bitbucket.org/2.0/users/%blabla%7D"
                  },
                  "html": {
                    "href": "https://bitbucket.org/%blabla%7D/"
                  },
                  "avatar": {
                    "href": "https://secure.gravatar.com/avatar/gfdgdg?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FSG-6.png"
                  }
                },
                "nickname": "repo1",
                "type": "user",
                "uuid": "{6485bd54-5433-45c5-a671-fc35b4166a49}"
              }
            },
            "summary": {
              "raw": ".gitignore created online with Bitbucket",
              "markup": "markdown",
              "html": "<p>.gitignore created online with Bitbucket</p>",
              "type": "rendered"
            },
            "parents": [],
            "date": "2019-10-16T12:15:54+00:00",
            "message": ".gitignore created online with Bitbucket",
            "type": "commit",
            "properties": {}
          }
        }
      }
    ]
  },
  "actor": {
    "display_name": "Vv",
    "account_id": "5b98adc8b2b15c2bdfcccf8a",
    "links": {
      "self": {
        "href": "https://api.bitbucket.org/2.0/users/%blabla%7D"
      },
      "html": {
        "href": "https://bitbucket.org/%blabla%7D/"
      },
      "avatar": {
        "href": "https://secure.gravatar.com/avatar/gfdg?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FSG-6.png"
      }
    },
    "nickname": "repo1",
    "type": "user",
    "uuid": "{6485bd54-5433-45c5-a671-fc35b4166a49}"
  },
  "repository": {
    "scm": "git",
    "website": null,
    "name": "testhook",
    "links": {
      "self": {
        "href": "https://api.bitbucket.org/2.0/repositories/repo1/testhook"
      },
      "html": {
        "href": "https://bitbucket.org/repo1/testhook"
      },
      "avatar": {
        "href": "https://bytebucket.org/ravatar/%7Bff5a2427-15ab-4b9e-96c5-023ce695e26f%7D?ts=default"
      }
    },
    "full_name": "repo1/testhook",
    "owner": {
      "display_name": "Vv",
      "account_id": "5b98adc8b2b15c2bdfcccf8a",
      "links": {
        "self": {
          "href": "https://api.bitbucket.org/2.0/users/%blabla%7D"
        },
        "html": {
          "href": "https://bitbucket.org/%blabla%7D/"
        },
        "avatar": {
          "href": "https://secure.gravatar.com/avatar/8c8a387e6330dff4cc43e1d594aca3c0?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FSG-6.png"
        }
      },
      "nickname": "repo1",
      "type": "user",
      "uuid": "{6485bd54-5433-45c5-a671-fc35b4166a49}"
    },
    "type": "repository",
    "is_private": true,
    "uuid": "{ff5a2427-15ab-4b9e-96c5-023ce695e26f}"
  }
}
`
