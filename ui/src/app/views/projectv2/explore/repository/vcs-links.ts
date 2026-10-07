import { VCSProject } from 'app/model/vcs.model';

/** Web pages of a repository on its vcs. */
export interface VCSWebLinks {
    repository: string;
    commit(sha: string): string;
    /** A full ref, `refs/heads/…` or `refs/tags/…`; null for any other. */
    ref(ref: string): string;
    pullRequest(id: number): string;
}

interface RefFormats {
    commit: (sha: string) => string;
    branch: (name: string) => string;
    tag: (name: string) => string;
    pullRequest: (id: number) => string;
}

/** Same formats as the vcs drivers of the engine (engine/vcs/<type>/client_repos.go), so that links match the runs ones. */
function formats(type: string, base: string): RefFormats {
    switch (type) {
        case 'github':
            return {
                commit: sha => `${base}/commit/${sha}`,
                branch: b => `${base}/commits/${encodeURI(b)}`,
                tag: t => `${base}/commits/${encodeURI(t)}`,
                pullRequest: id => `${base}/pull/${id}`
            };
        case 'gitlab':
            return {
                commit: sha => `${base}/-/commit/${sha}`,
                branch: b => `${base}/-/tree/${encodeURI(b)}?ref_type=heads`,
                tag: t => `${base}/-/tree/${encodeURI(t)}?ref_type=tags`,
                pullRequest: id => `${base}/-/merge_requests/${id}`
            };
        case 'forgejo':
            return {
                commit: sha => `${base}/commit/${sha}`,
                branch: b => `${base}/commits/branch/${encodeURI(b)}`,
                tag: t => `${base}/commits/tag/${encodeURI(t)}`,
                pullRequest: id => `${base}/pulls/${id}`
            };
        case 'bitbucketserver':
            return {
                commit: sha => `${base}/commits/${sha}`,
                branch: b => `${base}/browse?at=${encodeURIComponent('refs/heads/' + b)}`,
                tag: t => `${base}/browse?at=${encodeURIComponent('refs/tags/' + t)}`,
                pullRequest: id => `${base}/pull-requests/${id}`
            };
        default:
            return null;
    }
}

/** The web url of the repository: `<vcs>/<repo>`, but `<vcs>/projects/<key>/repos/<slug>` on Bitbucket Server. */
function repositoryURL(vcs: VCSProject, repoName: string): string {
    switch (vcs.type) {
        case 'bitbucketserver': {
            const [key, slug] = repoName.split('/', 2);
            return slug ? `${trimSlash(vcs.url)}/projects/${key}/repos/${slug}` : null;
        }
        default:
            return `${trimSlash(vcs.url)}/${repoName}`;
    }
}

function trimSlash(url: string): string {
    return url.replace(/\/+$/, '');
}

/** The web links of a repository, built from the vcs type and url without asking the vcs; null when they cannot be. */
export function vcsWebLinks(vcs: VCSProject, repoName: string): VCSWebLinks {
    if (!vcs?.url || !repoName) {
        return null;
    }
    const repository = repositoryURL(vcs, repoName);
    const f = repository ? formats(vcs.type, repository) : null;
    if (!f) {
        return null;
    }
    return {
        repository,
        commit: sha => sha ? f.commit(sha) : null,
        ref: ref => {
            if (ref?.startsWith('refs/heads/')) {
                return f.branch(ref.substring('refs/heads/'.length));
            }
            if (ref?.startsWith('refs/tags/')) {
                return f.tag(ref.substring('refs/tags/'.length));
            }
            return null;
        },
        pullRequest: id => id ? f.pullRequest(id) : null
    };
}
