import { VCSProject } from 'app/model/vcs.model';
import { vcsWebLinks } from './vcs-links';

const vcs = (type: string, url: string): VCSProject => <VCSProject>{ type, url };

describe('vcsWebLinks', () => {
    it('builds GitHub links', () => {
        const links = vcsWebLinks(vcs('github', 'https://github.com/'), 'ovh/cds');
        expect(links.repository).toEqual('https://github.com/ovh/cds');
        expect(links.commit('abc123')).toEqual('https://github.com/ovh/cds/commit/abc123');
        expect(links.ref('refs/heads/feat/x')).toEqual('https://github.com/ovh/cds/commits/feat/x');
        expect(links.ref('refs/tags/v1.0')).toEqual('https://github.com/ovh/cds/commits/v1.0');
        expect(links.pullRequest(42)).toEqual('https://github.com/ovh/cds/pull/42');
    });

    it('builds GitLab links', () => {
        const links = vcsWebLinks(vcs('gitlab', 'https://gitlab.com'), 'group/sub/project');
        expect(links.commit('abc123')).toEqual('https://gitlab.com/group/sub/project/-/commit/abc123');
        expect(links.ref('refs/heads/main')).toEqual('https://gitlab.com/group/sub/project/-/tree/main?ref_type=heads');
        expect(links.ref('refs/tags/v1.0')).toEqual('https://gitlab.com/group/sub/project/-/tree/v1.0?ref_type=tags');
        expect(links.pullRequest(42)).toEqual('https://gitlab.com/group/sub/project/-/merge_requests/42');
    });

    it('builds Forgejo links', () => {
        const links = vcsWebLinks(vcs('forgejo', 'https://git.example.com'), 'owner/repo');
        expect(links.commit('abc123')).toEqual('https://git.example.com/owner/repo/commit/abc123');
        expect(links.ref('refs/heads/main')).toEqual('https://git.example.com/owner/repo/commits/branch/main');
        expect(links.ref('refs/tags/v1.0')).toEqual('https://git.example.com/owner/repo/commits/tag/v1.0');
        expect(links.pullRequest(42)).toEqual('https://git.example.com/owner/repo/pulls/42');
    });

    it('builds Bitbucket Server links under projects/<key>/repos/<slug>', () => {
        const links = vcsWebLinks(vcs('bitbucketserver', 'https://stash.example.com'), 'PRJ/my-repo');
        expect(links.repository).toEqual('https://stash.example.com/projects/PRJ/repos/my-repo');
        expect(links.commit('abc123')).toEqual('https://stash.example.com/projects/PRJ/repos/my-repo/commits/abc123');
        expect(links.ref('refs/heads/feat/x')).toEqual('https://stash.example.com/projects/PRJ/repos/my-repo/browse?at=refs%2Fheads%2Ffeat%2Fx');
        expect(links.ref('refs/tags/v1.0')).toEqual('https://stash.example.com/projects/PRJ/repos/my-repo/browse?at=refs%2Ftags%2Fv1.0');
        expect(links.pullRequest(42)).toEqual('https://stash.example.com/projects/PRJ/repos/my-repo/pull-requests/42');
    });

    it('gives no link for an unknown ref, sha or pull request', () => {
        const links = vcsWebLinks(vcs('github', 'https://github.com'), 'ovh/cds');
        expect(links.ref('refs/pull/42/head')).toBeNull();
        expect(links.ref(null)).toBeNull();
        expect(links.commit('')).toBeNull();
        expect(links.pullRequest(0)).toBeNull();
    });

    it('gives no links when the vcs cannot tell them', () => {
        expect(vcsWebLinks(vcs('gitea', 'https://gitea.example.com'), 'owner/repo')).toBeNull();
        expect(vcsWebLinks(vcs('bitbucketcloud', 'https://bitbucket.org'), 'workspace/repo')).toBeNull();
        expect(vcsWebLinks(vcs('github', ''), 'ovh/cds')).toBeNull();
        expect(vcsWebLinks(vcs('bitbucketserver', 'https://stash.example.com'), 'no-key')).toBeNull();
        expect(vcsWebLinks(null, 'ovh/cds')).toBeNull();
    });
});
