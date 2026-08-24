export type EntityReferenceKind = 'action' | 'model';

/** A `uses:` or `runs-on:` value located in an entity definition. */
export class EntityReference {
    line: number;
    startColumn: number;
    endColumn: number;
    kind: EntityReferenceKind;
    /** Value as written, ref suffix included. */
    value: string;
}

/** Pieces of a CDS v2 entity path: <projectKey>/<vcsName>/<repoName>/<name>@<ref>. */
export class EntityPath {
    projectKey?: string;
    vcs?: string;
    repository?: string;
    name: string;
    ref?: string;
}

export class EntityReferenceUtils {
    private static readonly LINE = /^(\s*(?:-\s*)?(?:uses|runs-on):\s*)(\S.*?)\s*$/;

    /** Prefixes of a reference to a file of the current repository. */
    private static readonly LOCAL_PREFIXES = [
        '.cds/actions/',
        '.cds/worker-models/',
        '.cds/workflow-templates/'
    ];

    /** Locate the references of a YAML definition, with the columns of each value. */
    static scan(content: string): Array<EntityReference> {
        if (!content) {
            return [];
        }
        const references: Array<EntityReference> = [];
        content.split('\n').forEach((text, i) => {
            const match = text.match(EntityReferenceUtils.LINE);
            if (!match) {
                return;
            }
            references.push(<EntityReference>{
                line: i + 1,
                startColumn: match[1].length + 1,
                endColumn: match[1].length + match[2].length + 1,
                kind: match[1].indexOf('runs-on') !== -1 ? 'model' : 'action',
                value: match[2].replace(/^["']|["']$/g, '')
            });
        });
        return references;
    }

    /**
     * A repository name holds a slash, so the name is the last segment and the
     * repository is everything between the vcs and it. Returns null for values
     * that cannot denote a CDS entity, such as the `actions/checkout` plugins.
     *
     * `.cds/...` values point at a file of the current repository. The engine
     * resolves those by path and takes the entity name from the file content, which
     * is not readable from here, so the file name is used as a best guess.
     */
    static parse(raw: string): EntityPath {
        const at = raw.indexOf('@');
        const path = at === -1 ? raw : raw.substring(0, at);
        const ref = at === -1 ? null : raw.substring(at + 1);

        const localPrefix = EntityReferenceUtils.LOCAL_PREFIXES.find(p => path.startsWith(p));
        if (localPrefix) {
            const file = path.substring(localPrefix.length);
            const name = file.substring(file.lastIndexOf('/') + 1).replace(/\.ya?ml$/, '');
            return name ? <EntityPath>{ name, ref } : null;
        }

        const segments = path.split('/').filter(s => s.length > 0);
        if (segments.length === 1) {
            return <EntityPath>{ name: segments[0], ref };
        }
        if (segments.length >= 4) {
            return <EntityPath>{
                projectKey: segments[0],
                vcs: segments[1],
                repository: segments.slice(2, segments.length - 1).join('/'),
                name: segments[segments.length - 1],
                ref
            };
        }
        return null;
    }
}
