package github

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_filterCommits(t *testing.T) {
	commits := []Commit{}
	shas := []string{}
	json.Unmarshal([]byte(data), &commits)

	t.Logf("Getting commit between 595f1133abc43db6fcbb3e8aa42dba7d14b327c6 and ef37f16bab590c1cc8b4c897a0cef2820034ae81")
	commits = filterCommits(commits, "595f1133abc43db6fcbb3e8aa42dba7d14b327c6", "ef37f16bab590c1cc8b4c897a0cef2820034ae81")
	shas = []string{}
	for _, c := range commits {
		t.Logf(" - %s", c.Sha)
		shas = append(shas, c.Sha)
	}
	assert.EqualValues(t, []string{
		"ef37f16bab590c1cc8b4c897a0cef2820034ae81", "f3b129e9e6a468343f50ba9cd86192556db5bbfe", "055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef", "e991c2b082830fde76fb049da4d1b69269f3ff7c", "50fd96bac7b44263c24c5105d6fb4998a8c9069e", "75e604b5cf430b38b4cf99ccab13c2614991a811", "8ff2943b1e54d9356f2913c849d571af7fa9de78", "7573c5ce2d63bf8ebb5519ee1b001faaa693ed33"}, shas)

	t.Logf("Getting commit between 595f1133abc43db6fcbb3e8aa42dba7d14b327c6 and f3b129e9e6a468343f50ba9cd86192556db5bbfe")
	commits = filterCommits(commits, "595f1133abc43db6fcbb3e8aa42dba7d14b327c6", "f3b129e9e6a468343f50ba9cd86192556db5bbfe")
	shas = []string{}
	for _, c := range commits {
		t.Logf(" - %s", c.Sha)
		shas = append(shas, c.Sha)
	}
	assert.EqualValues(t, []string{
		"f3b129e9e6a468343f50ba9cd86192556db5bbfe", "055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef", "e991c2b082830fde76fb049da4d1b69269f3ff7c", "50fd96bac7b44263c24c5105d6fb4998a8c9069e", "75e604b5cf430b38b4cf99ccab13c2614991a811", "8ff2943b1e54d9356f2913c849d571af7fa9de78", "7573c5ce2d63bf8ebb5519ee1b001faaa693ed33"}, shas)

	t.Logf("Getting commit between 595f1133abc43db6fcbb3e8aa42dba7d14b327c6 and 055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef")
	commits = filterCommits(commits, "595f1133abc43db6fcbb3e8aa42dba7d14b327c6", "055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef")
	shas = []string{}
	for _, c := range commits {
		t.Logf(" - %s", c.Sha)
		shas = append(shas, c.Sha)
	}
	assert.EqualValues(t, []string{
		"055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef", "50fd96bac7b44263c24c5105d6fb4998a8c9069e", "75e604b5cf430b38b4cf99ccab13c2614991a811", "8ff2943b1e54d9356f2913c849d571af7fa9de78", "7573c5ce2d63bf8ebb5519ee1b001faaa693ed33"}, shas)

}

func Test_findAncestors(t *testing.T) {
	commits := []Commit{}
	json.Unmarshal([]byte(data), &commits)

	mapAncestors := map[string][]string{}
	for _, c := range commits {
		ancestors := findAncestors(commits, c.Sha)
		t.Logf("%s", c.Sha)
		for _, a := range ancestors {
			t.Logf("\t - %s", a)
		}
		mapAncestors[c.Sha] = ancestors
	}

	assert.EqualValues(t, []string{
		"2a6aa1e7dec80f8d6122286d21c771737fa07c85",
		"7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0",
		"fd7cbe1f5bc65a6e6267749f551fe894c834df1e",
		"a215fbdd289e37cbc08d050bd1d5c216dae108aa",
		"bfcb5f9b25b34c839db227e6bdfc424848e7fe70",
		"1be93df9618921773ae0631cfd2953ba2ab9b7f2",
		"d5cb0be315d53a77ff1020eaa77719c8a8ecc89c",
		"cf5c44d571c4eb740f9fc10165ff1b3d56610d7d",
		"329a2050b12b12ffde6048b51115b1d17dd7b37c",
		"a3f316fdb7c2e75a69e0ade5010485f1adadc1e4",
		"299eeb21d20b110279dca4e83d8c5b23c8a6ec59",
		"8e6f186c9611480ac26caf0b5f5d08ebdb0ad147",
		"0dad3035451526ad5ec56d35536d0ed53bf0b5b3",
		"8564a273c30f73151c259f3d65b285ae1da321bf",
		"642fcc07f856bda9dcc7ad72b70d178e36cb9f58",
		"fabb469a133d9ca6fa77ae2df58104690b0a4d4d",
		"b6cb7ae8babe1a6de8e2244d54b825a779a7367b",
		"8977de939702bbb75fc82bb8c6c0b50fe2bd4d40",
	},
		mapAncestors["a8b250edd3384b5ce8fd27dc33a7484140a5ef89"])

}

const data = `
[
  {
    "sha": "ef37f16bab590c1cc8b4c897a0cef2820034ae81",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-15T10:23:53Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-15T10:23:53Z"
      },
      "message": "fix (api): github cache",
      "tree": {
        "sha": "fea2fe0e9d005cdfa39ab31d4fd96df2053cc7c9",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/fea2fe0e9d005cdfa39ab31d4fd96df2053cc7c9"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/ef37f16bab590c1cc8b4c897a0cef2820034ae81",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/ef37f16bab590c1cc8b4c897a0cef2820034ae81",
    "html_url": "https://github.com/ovh/cds/commit/ef37f16bab590c1cc8b4c897a0cef2820034ae81",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/ef37f16bab590c1cc8b4c897a0cef2820034ae81/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "f3b129e9e6a468343f50ba9cd86192556db5bbfe",
        "url": "https://api.github.com/repos/ovh/cds/commits/f3b129e9e6a468343f50ba9cd86192556db5bbfe",
        "html_url": "https://github.com/ovh/cds/commit/f3b129e9e6a468343f50ba9cd86192556db5bbfe"
      }
    ]
  },
  {
    "sha": "f3b129e9e6a468343f50ba9cd86192556db5bbfe",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-15T08:26:07Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-15T08:26:07Z"
      },
      "message": "Merge branch 'master' into fsamin/fix\n\nConflicts:\n\tengine/api/repositoriesmanager/polling/repo_polling.go",
      "tree": {
        "sha": "e12c4dbb7bbb3da54fc858302f9b102f0d2900a5",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/e12c4dbb7bbb3da54fc858302f9b102f0d2900a5"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/f3b129e9e6a468343f50ba9cd86192556db5bbfe",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/f3b129e9e6a468343f50ba9cd86192556db5bbfe",
    "html_url": "https://github.com/ovh/cds/commit/f3b129e9e6a468343f50ba9cd86192556db5bbfe",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/f3b129e9e6a468343f50ba9cd86192556db5bbfe/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User", 
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef",
        "url": "https://api.github.com/repos/ovh/cds/commits/055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef",
        "html_url": "https://github.com/ovh/cds/commit/055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef"
      },
      {
        "sha": "e991c2b082830fde76fb049da4d1b69269f3ff7c",
        "url": "https://api.github.com/repos/ovh/cds/commits/e991c2b082830fde76fb049da4d1b69269f3ff7c",
        "html_url": "https://github.com/ovh/cds/commit/e991c2b082830fde76fb049da4d1b69269f3ff7c"
      }
    ]
  },
  {
    "sha": "055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-15T08:22:17Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-15T08:22:17Z"
      },
      "message": "fix (api): pipeline.CurrentAndPreviousPipelineBuildNumberAndHash",
      "tree": {
        "sha": "19b674b7fa8c9a14f0f3d8a17221632c3a0b7660",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/19b674b7fa8c9a14f0f3d8a17221632c3a0b7660"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef",
    "html_url": "https://github.com/ovh/cds/commit/055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/055b8ffb1678bda0f7eca1ff2ed32da5b4fb04ef/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "50fd96bac7b44263c24c5105d6fb4998a8c9069e",
        "url": "https://api.github.com/repos/ovh/cds/commits/50fd96bac7b44263c24c5105d6fb4998a8c9069e",
        "html_url": "https://github.com/ovh/cds/commit/50fd96bac7b44263c24c5105d6fb4998a8c9069e"
      }
    ]
  },
  {
    "sha": "e991c2b082830fde76fb049da4d1b69269f3ff7c",
    "commit": {
      "author": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-14T15:32:58Z"
      },
      "committer": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-12-14T15:32:58Z"
      },
      "message": "feat: improve handler return + update project modify date (#95)",
      "tree": {
        "sha": "f89ef6124d947d16b8803750845976342f2b328c",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/f89ef6124d947d16b8803750845976342f2b328c"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/e991c2b082830fde76fb049da4d1b69269f3ff7c",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/e991c2b082830fde76fb049da4d1b69269f3ff7c",
    "html_url": "https://github.com/ovh/cds/commit/e991c2b082830fde76fb049da4d1b69269f3ff7c",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/e991c2b082830fde76fb049da4d1b69269f3ff7c/comments",
    "author": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "595f1133abc43db6fcbb3e8aa42dba7d14b327c6",
        "url": "https://api.github.com/repos/ovh/cds/commits/595f1133abc43db6fcbb3e8aa42dba7d14b327c6",
        "html_url": "https://github.com/ovh/cds/commit/595f1133abc43db6fcbb3e8aa42dba7d14b327c6"
      }
    ]
  },
  {
    "sha": "50fd96bac7b44263c24c5105d6fb4998a8c9069e",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T15:31:16Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T15:31:16Z"
      },
      "message": "fix (api) : repo polling race condition",
      "tree": {
        "sha": "f811f2d570275d4ed8631d86ae0b4abe93e09780",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/f811f2d570275d4ed8631d86ae0b4abe93e09780"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/50fd96bac7b44263c24c5105d6fb4998a8c9069e",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/50fd96bac7b44263c24c5105d6fb4998a8c9069e",
    "html_url": "https://github.com/ovh/cds/commit/50fd96bac7b44263c24c5105d6fb4998a8c9069e",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/50fd96bac7b44263c24c5105d6fb4998a8c9069e/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "75e604b5cf430b38b4cf99ccab13c2614991a811",
        "url": "https://api.github.com/repos/ovh/cds/commits/75e604b5cf430b38b4cf99ccab13c2614991a811",
        "html_url": "https://github.com/ovh/cds/commit/75e604b5cf430b38b4cf99ccab13c2614991a811"
      }
    ]
  },
  {
    "sha": "75e604b5cf430b38b4cf99ccab13c2614991a811",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T15:15:52Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T15:15:52Z"
      },
      "message": "fix (api) : github commit list",
      "tree": {
        "sha": "78821d6e66b8aec8f4b372a8ba19da0822e50567",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/78821d6e66b8aec8f4b372a8ba19da0822e50567"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/75e604b5cf430b38b4cf99ccab13c2614991a811",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/75e604b5cf430b38b4cf99ccab13c2614991a811",
    "html_url": "https://github.com/ovh/cds/commit/75e604b5cf430b38b4cf99ccab13c2614991a811",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/75e604b5cf430b38b4cf99ccab13c2614991a811/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "8ff2943b1e54d9356f2913c849d571af7fa9de78",
        "url": "https://api.github.com/repos/ovh/cds/commits/8ff2943b1e54d9356f2913c849d571af7fa9de78",
        "html_url": "https://github.com/ovh/cds/commit/8ff2943b1e54d9356f2913c849d571af7fa9de78"
      }
    ]
  },
  {
    "sha": "595f1133abc43db6fcbb3e8aa42dba7d14b327c6",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-14T12:49:59Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-14T12:49:59Z"
      },
      "message": "fix (api): repository polling (#94)\n\n* fix (api): repository polling dead locks and monitoring\r\n* fix (api): repository polling clean goroutine typo",
      "tree": {
        "sha": "8b45245472684ef8d7b44fab119c2230b856b233",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/8b45245472684ef8d7b44fab119c2230b856b233"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/595f1133abc43db6fcbb3e8aa42dba7d14b327c6",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/595f1133abc43db6fcbb3e8aa42dba7d14b327c6",
    "html_url": "https://github.com/ovh/cds/commit/595f1133abc43db6fcbb3e8aa42dba7d14b327c6",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/595f1133abc43db6fcbb3e8aa42dba7d14b327c6/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "69d851f0c3605a424b253ac97bf7227ff377a45b",
        "url": "https://api.github.com/repos/ovh/cds/commits/69d851f0c3605a424b253ac97bf7227ff377a45b",
        "html_url": "https://github.com/ovh/cds/commit/69d851f0c3605a424b253ac97bf7227ff377a45b"
      }
    ]
  },
  {
    "sha": "8ff2943b1e54d9356f2913c849d571af7fa9de78",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T12:43:49Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T12:43:49Z"
      },
      "message": "fix (api): repository polling clean goroutine typo",
      "tree": {
        "sha": "8b45245472684ef8d7b44fab119c2230b856b233",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/8b45245472684ef8d7b44fab119c2230b856b233"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/8ff2943b1e54d9356f2913c849d571af7fa9de78",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/8ff2943b1e54d9356f2913c849d571af7fa9de78",
    "html_url": "https://github.com/ovh/cds/commit/8ff2943b1e54d9356f2913c849d571af7fa9de78",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/8ff2943b1e54d9356f2913c849d571af7fa9de78/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "7573c5ce2d63bf8ebb5519ee1b001faaa693ed33",
        "url": "https://api.github.com/repos/ovh/cds/commits/7573c5ce2d63bf8ebb5519ee1b001faaa693ed33",
        "html_url": "https://github.com/ovh/cds/commit/7573c5ce2d63bf8ebb5519ee1b001faaa693ed33"
      }
    ]
  },
  {
    "sha": "7573c5ce2d63bf8ebb5519ee1b001faaa693ed33",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T12:41:10Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin@corp.ovh.com",
        "date": "2016-12-14T12:41:35Z"
      },
      "message": "fix (api): repository polling dead locks and monitoring",
      "tree": {
        "sha": "042b80b30f0ee1e3c6102be036de48dd59a70a2b",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/042b80b30f0ee1e3c6102be036de48dd59a70a2b"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/7573c5ce2d63bf8ebb5519ee1b001faaa693ed33",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/7573c5ce2d63bf8ebb5519ee1b001faaa693ed33",
    "html_url": "https://github.com/ovh/cds/commit/7573c5ce2d63bf8ebb5519ee1b001faaa693ed33",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/7573c5ce2d63bf8ebb5519ee1b001faaa693ed33/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "69d851f0c3605a424b253ac97bf7227ff377a45b",
        "url": "https://api.github.com/repos/ovh/cds/commits/69d851f0c3605a424b253ac97bf7227ff377a45b",
        "html_url": "https://github.com/ovh/cds/commit/69d851f0c3605a424b253ac97bf7227ff377a45b"
      }
    ]
  },
  {
    "sha": "69d851f0c3605a424b253ac97bf7227ff377a45b",
    "commit": {
      "author": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-14T10:43:39Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-14T10:43:39Z"
      },
      "message": "feat (api): update project lastModified date + return project + env list (#93)",
      "tree": {
        "sha": "83887a33f0720a047a9854a7b67582ac2d31344a",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/83887a33f0720a047a9854a7b67582ac2d31344a"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/69d851f0c3605a424b253ac97bf7227ff377a45b",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/69d851f0c3605a424b253ac97bf7227ff377a45b",
    "html_url": "https://github.com/ovh/cds/commit/69d851f0c3605a424b253ac97bf7227ff377a45b",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/69d851f0c3605a424b253ac97bf7227ff377a45b/comments",
    "author": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "6d0d9f12808b134523563525967d38df165c2e13",
        "url": "https://api.github.com/repos/ovh/cds/commits/6d0d9f12808b134523563525967d38df165c2e13",
        "html_url": "https://github.com/ovh/cds/commit/6d0d9f12808b134523563525967d38df165c2e13"
      }
    ]
  },
  {
    "sha": "6d0d9f12808b134523563525967d38df165c2e13",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-14T08:52:30Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-14T08:52:30Z"
      },
      "message": "feat (api): mv scheduler package to queue package (#90)",
      "tree": {
        "sha": "2a1b3c5ddf46baff40bf5915b69b8c1d7a4913cd",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/2a1b3c5ddf46baff40bf5915b69b8c1d7a4913cd"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/6d0d9f12808b134523563525967d38df165c2e13",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/6d0d9f12808b134523563525967d38df165c2e13",
    "html_url": "https://github.com/ovh/cds/commit/6d0d9f12808b134523563525967d38df165c2e13",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/6d0d9f12808b134523563525967d38df165c2e13/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "9ae2123af163ecd3fd0a6c52c86bbd66479df05f",
        "url": "https://api.github.com/repos/ovh/cds/commits/9ae2123af163ecd3fd0a6c52c86bbd66479df05f",
        "html_url": "https://github.com/ovh/cds/commit/9ae2123af163ecd3fd0a6c52c86bbd66479df05f"
      }
    ]
  },
  {
    "sha": "9ae2123af163ecd3fd0a6c52c86bbd66479df05f",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-13T20:17:55Z"
      },
      "committer": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-12-13T20:17:55Z"
      },
      "message": "fix (worker, api): worker export variable on deploy pipeline (#92)",
      "tree": {
        "sha": "f5d2599a1f0684d14134be6fd77c6935c2e4f4ce",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/f5d2599a1f0684d14134be6fd77c6935c2e4f4ce"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/9ae2123af163ecd3fd0a6c52c86bbd66479df05f",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/9ae2123af163ecd3fd0a6c52c86bbd66479df05f",
    "html_url": "https://github.com/ovh/cds/commit/9ae2123af163ecd3fd0a6c52c86bbd66479df05f",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/9ae2123af163ecd3fd0a6c52c86bbd66479df05f/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "a8b250edd3384b5ce8fd27dc33a7484140a5ef89",
        "url": "https://api.github.com/repos/ovh/cds/commits/a8b250edd3384b5ce8fd27dc33a7484140a5ef89",
        "html_url": "https://github.com/ovh/cds/commit/a8b250edd3384b5ce8fd27dc33a7484140a5ef89"
      }
    ]
  },
  {
    "sha": "a8b250edd3384b5ce8fd27dc33a7484140a5ef89",
    "commit": {
      "author": {
        "name": "Pierre Roullon",
        "email": "proullon@users.noreply.github.com",
        "date": "2016-12-13T20:17:33Z"
      },
      "committer": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-12-13T20:17:33Z"
      },
      "message": "fixes in api and hatchery cli (#91)\n\n* fix (database migration cli): error output\r\n\r\n* fix (hatchery): fix behavior of parameters lookup in env",
      "tree": {
        "sha": "bd48274c037bdef0b5b5de37a58fa49a9696d709",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/bd48274c037bdef0b5b5de37a58fa49a9696d709"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/a8b250edd3384b5ce8fd27dc33a7484140a5ef89",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/a8b250edd3384b5ce8fd27dc33a7484140a5ef89",
    "html_url": "https://github.com/ovh/cds/commit/a8b250edd3384b5ce8fd27dc33a7484140a5ef89",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/a8b250edd3384b5ce8fd27dc33a7484140a5ef89/comments",
    "author": {
      "login": "proullon",
      "id": 3083356,
      "avatar_url": "https://avatars.githubusercontent.com/u/3083356?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/proullon",
      "html_url": "https://github.com/proullon",
      "followers_url": "https://api.github.com/users/proullon/followers",
      "following_url": "https://api.github.com/users/proullon/following{/other_user}",
      "gists_url": "https://api.github.com/users/proullon/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/proullon/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/proullon/subscriptions",
      "organizations_url": "https://api.github.com/users/proullon/orgs",
      "repos_url": "https://api.github.com/users/proullon/repos",
      "events_url": "https://api.github.com/users/proullon/events{/privacy}",
      "received_events_url": "https://api.github.com/users/proullon/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "2a6aa1e7dec80f8d6122286d21c771737fa07c85",
        "url": "https://api.github.com/repos/ovh/cds/commits/2a6aa1e7dec80f8d6122286d21c771737fa07c85",
        "html_url": "https://github.com/ovh/cds/commit/2a6aa1e7dec80f8d6122286d21c771737fa07c85"
      }
    ]
  },
  {
    "sha": "2a6aa1e7dec80f8d6122286d21c771737fa07c85",
    "commit": {
      "author": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-12T22:00:04Z"
      },
      "committer": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-12-12T22:00:04Z"
      },
      "message": "feat : load artifacts and tests with build result (#89)\n\n* feat : load artifacts and tests with build result\r\n\r\n* fix: code review, add omitempty",
      "tree": {
        "sha": "edcc8cce9733fdf199868132848fca975d1418a6",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/edcc8cce9733fdf199868132848fca975d1418a6"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/2a6aa1e7dec80f8d6122286d21c771737fa07c85",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/2a6aa1e7dec80f8d6122286d21c771737fa07c85",
    "html_url": "https://github.com/ovh/cds/commit/2a6aa1e7dec80f8d6122286d21c771737fa07c85",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/2a6aa1e7dec80f8d6122286d21c771737fa07c85/comments",
    "author": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0",
        "url": "https://api.github.com/repos/ovh/cds/commits/7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0",
        "html_url": "https://github.com/ovh/cds/commit/7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0"
      }
    ]
  },
  {
    "sha": "7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-09T08:31:29Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-09T08:31:29Z"
      },
      "message": "fix (api): add OwnerID on model struct to be able to scan old fashioned database (#88)",
      "tree": {
        "sha": "e38242f57caac829f53f034d1f2b6fd29836c342",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/e38242f57caac829f53f034d1f2b6fd29836c342"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0",
    "html_url": "https://github.com/ovh/cds/commit/7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/7ef75d97322c25faecaa1a7f7e8f730a3f45fbd0/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "fd7cbe1f5bc65a6e6267749f551fe894c834df1e",
        "url": "https://api.github.com/repos/ovh/cds/commits/fd7cbe1f5bc65a6e6267749f551fe894c834df1e",
        "html_url": "https://github.com/ovh/cds/commit/fd7cbe1f5bc65a6e6267749f551fe894c834df1e"
      }
    ]
  },
  {
    "sha": "fd7cbe1f5bc65a6e6267749f551fe894c834df1e",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-07T08:53:02Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-07T08:53:02Z"
      },
      "message": "fix (sql): migration file #3 (#87)",
      "tree": {
        "sha": "ae7bcc789cb7be1bf3d60afc5ec356249d5687f2",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/ae7bcc789cb7be1bf3d60afc5ec356249d5687f2"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/fd7cbe1f5bc65a6e6267749f551fe894c834df1e",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/fd7cbe1f5bc65a6e6267749f551fe894c834df1e",
    "html_url": "https://github.com/ovh/cds/commit/fd7cbe1f5bc65a6e6267749f551fe894c834df1e",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/fd7cbe1f5bc65a6e6267749f551fe894c834df1e/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "a215fbdd289e37cbc08d050bd1d5c216dae108aa",
        "url": "https://api.github.com/repos/ovh/cds/commits/a215fbdd289e37cbc08d050bd1d5c216dae108aa",
        "html_url": "https://github.com/ovh/cds/commit/a215fbdd289e37cbc08d050bd1d5c216dae108aa"
      }
    ]
  },
  {
    "sha": "a215fbdd289e37cbc08d050bd1d5c216dae108aa",
    "commit": {
      "author": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-07T08:27:14Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-07T08:27:14Z"
      },
      "message": "fix (api): do not disabled building worker (#86)",
      "tree": {
        "sha": "a7e98dc78876820210171ed0dcb2ac1d683e6acb",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/a7e98dc78876820210171ed0dcb2ac1d683e6acb"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/a215fbdd289e37cbc08d050bd1d5c216dae108aa",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/a215fbdd289e37cbc08d050bd1d5c216dae108aa",
    "html_url": "https://github.com/ovh/cds/commit/a215fbdd289e37cbc08d050bd1d5c216dae108aa",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/a215fbdd289e37cbc08d050bd1d5c216dae108aa/comments",
    "author": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "bfcb5f9b25b34c839db227e6bdfc424848e7fe70",
        "url": "https://api.github.com/repos/ovh/cds/commits/bfcb5f9b25b34c839db227e6bdfc424848e7fe70",
        "html_url": "https://github.com/ovh/cds/commit/bfcb5f9b25b34c839db227e6bdfc424848e7fe70"
      }
    ]
  },
  {
    "sha": "bfcb5f9b25b34c839db227e6bdfc424848e7fe70",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-05T13:27:05Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-05T13:27:05Z"
      },
      "message": "fix (api): stage prerequisites (#84)",
      "tree": {
        "sha": "b25341f68074c61e7dacd7b501b7a278227ecdfa",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/b25341f68074c61e7dacd7b501b7a278227ecdfa"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/bfcb5f9b25b34c839db227e6bdfc424848e7fe70",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/bfcb5f9b25b34c839db227e6bdfc424848e7fe70",
    "html_url": "https://github.com/ovh/cds/commit/bfcb5f9b25b34c839db227e6bdfc424848e7fe70",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/bfcb5f9b25b34c839db227e6bdfc424848e7fe70/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "1be93df9618921773ae0631cfd2953ba2ab9b7f2",
        "url": "https://api.github.com/repos/ovh/cds/commits/1be93df9618921773ae0631cfd2953ba2ab9b7f2",
        "html_url": "https://github.com/ovh/cds/commit/1be93df9618921773ae0631cfd2953ba2ab9b7f2"
      }
    ]
  },
  {
    "sha": "1be93df9618921773ae0631cfd2953ba2ab9b7f2",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-05T13:26:27Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-05T13:26:27Z"
      },
      "message": "fix (sql): clean worker model migration script (#85)",
      "tree": {
        "sha": "a277065b56a383b01df34301d41c2786ec969d80",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/a277065b56a383b01df34301d41c2786ec969d80"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/1be93df9618921773ae0631cfd2953ba2ab9b7f2",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/1be93df9618921773ae0631cfd2953ba2ab9b7f2",
    "html_url": "https://github.com/ovh/cds/commit/1be93df9618921773ae0631cfd2953ba2ab9b7f2",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/1be93df9618921773ae0631cfd2953ba2ab9b7f2/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "d5cb0be315d53a77ff1020eaa77719c8a8ecc89c",
        "url": "https://api.github.com/repos/ovh/cds/commits/d5cb0be315d53a77ff1020eaa77719c8a8ecc89c",
        "html_url": "https://github.com/ovh/cds/commit/d5cb0be315d53a77ff1020eaa77719c8a8ecc89c"
      }
    ]
  },
  {
    "sha": "d5cb0be315d53a77ff1020eaa77719c8a8ecc89c",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-02T13:47:55Z"
      },
      "committer": {
        "name": "GitHub",
        "email": "noreply@github.com",
        "date": "2016-12-02T13:47:55Z"
      },
      "message": "feat (api): router not found handler + logs (#83)",
      "tree": {
        "sha": "7d7383cc202459f42f75e98ae540cd1b5271791d",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/7d7383cc202459f42f75e98ae540cd1b5271791d"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/d5cb0be315d53a77ff1020eaa77719c8a8ecc89c",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/d5cb0be315d53a77ff1020eaa77719c8a8ecc89c",
    "html_url": "https://github.com/ovh/cds/commit/d5cb0be315d53a77ff1020eaa77719c8a8ecc89c",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/d5cb0be315d53a77ff1020eaa77719c8a8ecc89c/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "web-flow",
      "id": 19864447,
      "avatar_url": "https://avatars.githubusercontent.com/u/19864447?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/web-flow",
      "html_url": "https://github.com/web-flow",
      "followers_url": "https://api.github.com/users/web-flow/followers",
      "following_url": "https://api.github.com/users/web-flow/following{/other_user}",
      "gists_url": "https://api.github.com/users/web-flow/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/web-flow/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/web-flow/subscriptions",
      "organizations_url": "https://api.github.com/users/web-flow/orgs",
      "repos_url": "https://api.github.com/users/web-flow/repos",
      "events_url": "https://api.github.com/users/web-flow/events{/privacy}",
      "received_events_url": "https://api.github.com/users/web-flow/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "cf5c44d571c4eb740f9fc10165ff1b3d56610d7d",
        "url": "https://api.github.com/repos/ovh/cds/commits/cf5c44d571c4eb740f9fc10165ff1b3d56610d7d",
        "html_url": "https://github.com/ovh/cds/commit/cf5c44d571c4eb740f9fc10165ff1b3d56610d7d"
      }
    ]
  },
  {
    "sha": "cf5c44d571c4eb740f9fc10165ff1b3d56610d7d",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-02T10:17:54Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-02T10:17:54Z"
      },
      "message": "fix (test): missing import (#82)",
      "tree": {
        "sha": "628739a8236cf23a709cc45050550058450a4576",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/628739a8236cf23a709cc45050550058450a4576"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/cf5c44d571c4eb740f9fc10165ff1b3d56610d7d",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/cf5c44d571c4eb740f9fc10165ff1b3d56610d7d",
    "html_url": "https://github.com/ovh/cds/commit/cf5c44d571c4eb740f9fc10165ff1b3d56610d7d",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/cf5c44d571c4eb740f9fc10165ff1b3d56610d7d/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "329a2050b12b12ffde6048b51115b1d17dd7b37c",
        "url": "https://api.github.com/repos/ovh/cds/commits/329a2050b12b12ffde6048b51115b1d17dd7b37c",
        "html_url": "https://github.com/ovh/cds/commit/329a2050b12b12ffde6048b51115b1d17dd7b37c"
      }
    ]
  },
  {
    "sha": "329a2050b12b12ffde6048b51115b1d17dd7b37c",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-12-02T09:23:35Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-02T09:23:35Z"
      },
      "message": "feat (api/hatchery/worker): worker model permission (#80)\n\n* feat (hatchery/worker): worker model permission\r\n* feat (api): worker model persmission management",
      "tree": {
        "sha": "4753dcc09e346c8b138179337bf9f538da97f3f7",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/4753dcc09e346c8b138179337bf9f538da97f3f7"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/329a2050b12b12ffde6048b51115b1d17dd7b37c",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/329a2050b12b12ffde6048b51115b1d17dd7b37c",
    "html_url": "https://github.com/ovh/cds/commit/329a2050b12b12ffde6048b51115b1d17dd7b37c",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/329a2050b12b12ffde6048b51115b1d17dd7b37c/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "a3f316fdb7c2e75a69e0ade5010485f1adadc1e4",
        "url": "https://api.github.com/repos/ovh/cds/commits/a3f316fdb7c2e75a69e0ade5010485f1adadc1e4",
        "html_url": "https://github.com/ovh/cds/commit/a3f316fdb7c2e75a69e0ade5010485f1adadc1e4"
      }
    ]
  },
  {
    "sha": "a3f316fdb7c2e75a69e0ade5010485f1adadc1e4",
    "commit": {
      "author": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-12-01T09:01:43Z"
      },
      "committer": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-12-01T09:01:43Z"
      },
      "message": "feat: allow adding a trigger in pipeline child (and not in parent) (#81)",
      "tree": {
        "sha": "df55ba37a95895d4aaa740cbe47a1ae158ea1de3",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/df55ba37a95895d4aaa740cbe47a1ae158ea1de3"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/a3f316fdb7c2e75a69e0ade5010485f1adadc1e4",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/a3f316fdb7c2e75a69e0ade5010485f1adadc1e4",
    "html_url": "https://github.com/ovh/cds/commit/a3f316fdb7c2e75a69e0ade5010485f1adadc1e4",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/a3f316fdb7c2e75a69e0ade5010485f1adadc1e4/comments",
    "author": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "299eeb21d20b110279dca4e83d8c5b23c8a6ec59",
        "url": "https://api.github.com/repos/ovh/cds/commits/299eeb21d20b110279dca4e83d8c5b23c8a6ec59",
        "html_url": "https://github.com/ovh/cds/commit/299eeb21d20b110279dca4e83d8c5b23c8a6ec59"
      }
    ]
  },
  {
    "sha": "299eeb21d20b110279dca4e83d8c5b23c8a6ec59",
    "commit": {
      "author": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-11-29T20:41:19Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-11-29T20:41:19Z"
      },
      "message": "feat (api): get application with trigger handler (#79)\n\n* add get application with trigger\r\n\r\n* rename function + add error management\r\n\r\n* fix doc\r\n\r\n* feat: add handler test\r\n\r\n* fix: remove import",
      "tree": {
        "sha": "6ec50ded0cce45c1c122598b527d33ad108bf6b7",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/6ec50ded0cce45c1c122598b527d33ad108bf6b7"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/299eeb21d20b110279dca4e83d8c5b23c8a6ec59",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/299eeb21d20b110279dca4e83d8c5b23c8a6ec59",
    "html_url": "https://github.com/ovh/cds/commit/299eeb21d20b110279dca4e83d8c5b23c8a6ec59",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/299eeb21d20b110279dca4e83d8c5b23c8a6ec59/comments",
    "author": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "8e6f186c9611480ac26caf0b5f5d08ebdb0ad147",
        "url": "https://api.github.com/repos/ovh/cds/commits/8e6f186c9611480ac26caf0b5f5d08ebdb0ad147",
        "html_url": "https://github.com/ovh/cds/commit/8e6f186c9611480ac26caf0b5f5d08ebdb0ad147"
      }
    ]
  },
  {
    "sha": "8e6f186c9611480ac26caf0b5f5d08ebdb0ad147",
    "commit": {
      "author": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-11-29T20:40:24Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-11-29T20:40:24Z"
      },
      "message": "feat (api): refactor keys & default value to enabled attr in step : true (#78)\n\n* fix: code review\r\n\r\n* feat: actionName not mandatory with --url arg\r\n\r\n* fix: worker_model.owner_id nullabled\r\n\r\n* feat: add log warn\r\n\r\n* fix: remove first line -h doc\r\n\r\n* feat: :%s/loadAction/importAction/g\r\n\r\n* feat: worker upload --tag=<tag> path\r\n\r\n* fix: code review\r\n\r\n* fix: code review\r\n\r\n* fix: timeout on request\r\n\r\n* feat: default value to enabled attr in step : true\r\n\r\n* feat: refactor keys package\r\n\r\n* feat: test on default values\r\n\r\n* fix: default value with existing action on apply template\r\n\r\nSigned-off-by: Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>",
      "tree": {
        "sha": "0cec88cbe6f7793a9bcb4af73852e377e3bf2b2c",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/0cec88cbe6f7793a9bcb4af73852e377e3bf2b2c"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/8e6f186c9611480ac26caf0b5f5d08ebdb0ad147",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/8e6f186c9611480ac26caf0b5f5d08ebdb0ad147",
    "html_url": "https://github.com/ovh/cds/commit/8e6f186c9611480ac26caf0b5f5d08ebdb0ad147",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/8e6f186c9611480ac26caf0b5f5d08ebdb0ad147/comments",
    "author": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "0dad3035451526ad5ec56d35536d0ed53bf0b5b3",
        "url": "https://api.github.com/repos/ovh/cds/commits/0dad3035451526ad5ec56d35536d0ed53bf0b5b3",
        "html_url": "https://github.com/ovh/cds/commit/0dad3035451526ad5ec56d35536d0ed53bf0b5b3"
      }
    ]
  },
  {
    "sha": "0dad3035451526ad5ec56d35536d0ed53bf0b5b3",
    "commit": {
      "author": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-11-28T10:35:52Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-11-28T10:35:52Z"
      },
      "message": "feat (api): add flags final / enabled on action import HCL (#77)",
      "tree": {
        "sha": "d113d9045e359cf680c955dec73f9fb076d097f7",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/d113d9045e359cf680c955dec73f9fb076d097f7"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/0dad3035451526ad5ec56d35536d0ed53bf0b5b3",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/0dad3035451526ad5ec56d35536d0ed53bf0b5b3",
    "html_url": "https://github.com/ovh/cds/commit/0dad3035451526ad5ec56d35536d0ed53bf0b5b3",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/0dad3035451526ad5ec56d35536d0ed53bf0b5b3/comments",
    "author": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "8564a273c30f73151c259f3d65b285ae1da321bf",
        "url": "https://api.github.com/repos/ovh/cds/commits/8564a273c30f73151c259f3d65b285ae1da321bf",
        "html_url": "https://github.com/ovh/cds/commit/8564a273c30f73151c259f3d65b285ae1da321bf"
      }
    ]
  },
  {
    "sha": "8564a273c30f73151c259f3d65b285ae1da321bf",
    "commit": {
      "author": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-11-28T10:25:41Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-11-28T10:25:41Z"
      },
      "message": "fix (api): return updated pipeline (#76)",
      "tree": {
        "sha": "ab79b3c5e6f172a3287d37ba7479a990336adac9",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/ab79b3c5e6f172a3287d37ba7479a990336adac9"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/8564a273c30f73151c259f3d65b285ae1da321bf",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/8564a273c30f73151c259f3d65b285ae1da321bf",
    "html_url": "https://github.com/ovh/cds/commit/8564a273c30f73151c259f3d65b285ae1da321bf",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/8564a273c30f73151c259f3d65b285ae1da321bf/comments",
    "author": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "642fcc07f856bda9dcc7ad72b70d178e36cb9f58",
        "url": "https://api.github.com/repos/ovh/cds/commits/642fcc07f856bda9dcc7ad72b70d178e36cb9f58",
        "html_url": "https://github.com/ovh/cds/commit/642fcc07f856bda9dcc7ad72b70d178e36cb9f58"
      }
    ]
  },
  {
    "sha": "642fcc07f856bda9dcc7ad72b70d178e36cb9f58",
    "commit": {
      "author": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-11-25T08:31:57Z"
      },
      "committer": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-11-25T08:31:57Z"
      },
      "message": "feat (cli): move dir (#75)\n\n* feat (cli): move dir\r\n\r\n* fix (cli): readme",
      "tree": {
        "sha": "efd68f7bb952ed9204f645240bf755b03758cb83",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/efd68f7bb952ed9204f645240bf755b03758cb83"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/642fcc07f856bda9dcc7ad72b70d178e36cb9f58",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/642fcc07f856bda9dcc7ad72b70d178e36cb9f58",
    "html_url": "https://github.com/ovh/cds/commit/642fcc07f856bda9dcc7ad72b70d178e36cb9f58",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/642fcc07f856bda9dcc7ad72b70d178e36cb9f58/comments",
    "author": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "fabb469a133d9ca6fa77ae2df58104690b0a4d4d",
        "url": "https://api.github.com/repos/ovh/cds/commits/fabb469a133d9ca6fa77ae2df58104690b0a4d4d",
        "html_url": "https://github.com/ovh/cds/commit/fabb469a133d9ca6fa77ae2df58104690b0a4d4d"
      }
    ]
  },
  {
    "sha": "fabb469a133d9ca6fa77ae2df58104690b0a4d4d",
    "commit": {
      "author": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-11-24T16:45:57Z"
      },
      "committer": {
        "name": "François Samin",
        "email": "francois.samin+github@gmail.com",
        "date": "2016-11-24T16:45:57Z"
      },
      "message": "feat (worker): worker upload --tag=<tag> path (#74)\n\n* feat: worker upload --tag=<tag> path",
      "tree": {
        "sha": "9dd66438e98f076f6621c07aa14857a7ea4f39f8",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/9dd66438e98f076f6621c07aa14857a7ea4f39f8"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/fabb469a133d9ca6fa77ae2df58104690b0a4d4d",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/fabb469a133d9ca6fa77ae2df58104690b0a4d4d",
    "html_url": "https://github.com/ovh/cds/commit/fabb469a133d9ca6fa77ae2df58104690b0a4d4d",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/fabb469a133d9ca6fa77ae2df58104690b0a4d4d/comments",
    "author": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "fsamin",
      "id": 684151,
      "avatar_url": "https://avatars.githubusercontent.com/u/684151?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/fsamin",
      "html_url": "https://github.com/fsamin",
      "followers_url": "https://api.github.com/users/fsamin/followers",
      "following_url": "https://api.github.com/users/fsamin/following{/other_user}",
      "gists_url": "https://api.github.com/users/fsamin/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/fsamin/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/fsamin/subscriptions",
      "organizations_url": "https://api.github.com/users/fsamin/orgs",
      "repos_url": "https://api.github.com/users/fsamin/repos",
      "events_url": "https://api.github.com/users/fsamin/events{/privacy}",
      "received_events_url": "https://api.github.com/users/fsamin/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "b6cb7ae8babe1a6de8e2244d54b825a779a7367b",
        "url": "https://api.github.com/repos/ovh/cds/commits/b6cb7ae8babe1a6de8e2244d54b825a779a7367b",
        "html_url": "https://github.com/ovh/cds/commit/b6cb7ae8babe1a6de8e2244d54b825a779a7367b"
      }
    ]
  },
  {
    "sha": "b6cb7ae8babe1a6de8e2244d54b825a779a7367b",
    "commit": {
      "author": {
        "name": "Guiheux Steven",
        "email": "steven.guiheux+github@gmail.com",
        "date": "2016-11-24T15:24:22Z"
      },
      "committer": {
        "name": "Yvonnick Esnault",
        "email": "yesnault@users.noreply.github.com",
        "date": "2016-11-24T15:24:22Z"
      },
      "message": "feat: return pipeline updated after add/update/delete pipeline parameters (#73)",
      "tree": {
        "sha": "fa1e3aeed9e02a8ab7af0abaf1ad7c2c31cfcf0a",
        "url": "https://api.github.com/repos/ovh/cds/git/trees/fa1e3aeed9e02a8ab7af0abaf1ad7c2c31cfcf0a"
      },
      "url": "https://api.github.com/repos/ovh/cds/git/commits/b6cb7ae8babe1a6de8e2244d54b825a779a7367b",
      "comment_count": 0
    },
    "url": "https://api.github.com/repos/ovh/cds/commits/b6cb7ae8babe1a6de8e2244d54b825a779a7367b",
    "html_url": "https://github.com/ovh/cds/commit/b6cb7ae8babe1a6de8e2244d54b825a779a7367b",
    "comments_url": "https://api.github.com/repos/ovh/cds/commits/b6cb7ae8babe1a6de8e2244d54b825a779a7367b/comments",
    "author": {
      "login": "sguiheux",
      "id": 1478025,
      "avatar_url": "https://avatars.githubusercontent.com/u/1478025?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sguiheux",
      "html_url": "https://github.com/sguiheux",
      "followers_url": "https://api.github.com/users/sguiheux/followers",
      "following_url": "https://api.github.com/users/sguiheux/following{/other_user}",
      "gists_url": "https://api.github.com/users/sguiheux/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sguiheux/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sguiheux/subscriptions",
      "organizations_url": "https://api.github.com/users/sguiheux/orgs",
      "repos_url": "https://api.github.com/users/sguiheux/repos",
      "events_url": "https://api.github.com/users/sguiheux/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sguiheux/received_events",
      "type": "User",
      "site_admin": false
    },
    "committer": {
      "login": "yesnault",
      "id": 395454,
      "avatar_url": "https://avatars.githubusercontent.com/u/395454?v=3",
      "gravatar_id": "",
      "url": "https://api.github.com/users/yesnault",
      "html_url": "https://github.com/yesnault",
      "followers_url": "https://api.github.com/users/yesnault/followers",
      "following_url": "https://api.github.com/users/yesnault/following{/other_user}",
      "gists_url": "https://api.github.com/users/yesnault/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/yesnault/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/yesnault/subscriptions",
      "organizations_url": "https://api.github.com/users/yesnault/orgs",
      "repos_url": "https://api.github.com/users/yesnault/repos",
      "events_url": "https://api.github.com/users/yesnault/events{/privacy}",
      "received_events_url": "https://api.github.com/users/yesnault/received_events",
      "type": "User",
      "site_admin": false
    },
    "parents": [
      {
        "sha": "8977de939702bbb75fc82bb8c6c0b50fe2bd4d40",
        "url": "https://api.github.com/repos/ovh/cds/commits/8977de939702bbb75fc82bb8c6c0b50fe2bd4d40",
        "html_url": "https://github.com/ovh/cds/commit/8977de939702bbb75fc82bb8c6c0b50fe2bd4d40"
      }
    ]
  }
]
`
