import { inject, Injectable } from '@angular/core';
import { Store } from '@ngxs/store';
import { BehaviorSubject, lastValueFrom } from 'rxjs';
import { Project, ProjectDistantRepositoryWorkflow, ProjectRepository, RepositoryHookEvent } from 'app/model/project.model';
import { VCSProject } from 'app/model/vcs.model';
import { Branch, Tag } from 'app/model/repositories.model';
import { RepositoryAnalysis } from 'app/model/analysis.model';
import { Entity, EntityType } from 'app/model/entity.model';
import { ProjectService } from 'app/service/project/project.service';
import { PreferencesState } from 'app/store/preferences.state';
import * as actionPreferences from 'app/store/preferences.action';
import { ProjectV2State } from 'app/store/project-v2.state';
import { groupEntities } from './entities';

/** `master` for `refs/heads/master`, `v1.0` for `refs/tags/v1.0`. */
export function shortRef(ref: string): string {
    return (ref ?? '').replace(/^refs\/(heads|tags)\//, '');
}

/**
 * State of the repository page, shared by its header and its tabs. The tabs live behind a
 * router-outlet and cannot receive it as inputs; the page component provides one instance
 * of this service, which lives as long as the page does.
 */
@Injectable()
export class RepositoryContextService {
    readonly vcs$ = new BehaviorSubject<VCSProject>(null);
    readonly repository$ = new BehaviorSubject<ProjectRepository>(null);
    readonly branches$ = new BehaviorSubject<Array<Branch>>([]);
    readonly tags$ = new BehaviorSubject<Array<Tag>>([]);
    readonly ref$ = new BehaviorSubject<string>(null);
    // Lists are null until read, so that a consumer can tell "not yet" from "none"
    readonly analyses$ = new BehaviorSubject<Array<RepositoryAnalysis>>(null);
    readonly events$ = new BehaviorSubject<Array<RepositoryHookEvent>>(null);
    /** The entities of the current ref, by type. */
    readonly entities$ = new BehaviorSubject<Map<EntityType, Array<Entity>>>(null);
    /** Why the events could not be read, when they could not; the page shows it instead of failing. */
    readonly eventsError$ = new BehaviorSubject<any>(null);
    /** For a listened repository, the workflows of the project that listen to it; null until read. */
    readonly listenedBy$ = new BehaviorSubject<Array<ProjectDistantRepositoryWorkflow>>(null);

    private _project: Project;
    private _vcsName: string;
    private _repoName: string;
    // Incremented by each load; a load whose id is no longer current drops what it read
    private _loadId = 0;

    private _projectService = inject(ProjectService);
    private _store = inject(Store);

    get project(): Project { return this._project; }
    get vcsName(): string { return this._vcsName; }
    get repoName(): string { return this._repoName; }
    /** `vcs/repository`, the identifier the drawers and the preferences use. */
    get repositoryPath(): string { return `${this._vcsName}/${this._repoName}`; }
    get ref(): string { return this.ref$.value; }

    /**
     * Reads what the page needs for the repository the url names. Called again for the same
     * repository, it only applies the ref.
     */
    async load(vcsName: string, repoName: string, refFromUrl: string): Promise<void> {
        if (this._vcsName === vcsName && this._repoName === repoName) {
            await this.selectRef(refFromUrl);
            return;
        }
        const loadId = ++this._loadId;
        this._project = this._store.selectSnapshot(ProjectV2State.current);
        this._vcsName = vcsName;
        this._repoName = repoName;
        this.vcs$.next(null);
        this.repository$.next(null);
        this.branches$.next([]);
        this.tags$.next([]);
        this.ref$.next(null);
        this.entities$.next(null);
        this.analyses$.next(null);
        this.events$.next(null);
        this.eventsError$.next(null);
        this.listenedBy$.next(null);

        const key = this._project.key;
        const [vcs, repository] = await Promise.all([
            lastValueFrom(this._projectService.getVCSProject(key, vcsName)),
            lastValueFrom(this._projectService.getVCSRepository(key, vcsName, repoName)).catch(e => {
                // A repository the project only listens to is not declared: nothing but its name is known
                if (e?.status === 404) {
                    return <ProjectRepository>{ name: repoName, distant: true };
                }
                throw e;
            })
        ]);
        if (loadId !== this._loadId) {
            return;
        }
        this.vcs$.next(vcs);
        this.repository$.next(repository);

        // Events are keyed by names, so a listened repository has them too; the rest needs a declared one
        const events = this.readEvents(loadId);
        if (repository.distant) {
            const [distant] = await Promise.all([
                lastValueFrom(this._projectService.getDistantRepositories(key)),
                events
            ]);
            if (loadId !== this._loadId) {
                return;
            }
            // The API keys listened repositories by lowercased name
            const listened = distant.find(d => d.vcs_name === vcsName && d.repository === repoName.toLowerCase());
            this.listenedBy$.next(listened?.workflows ?? []);
            return;
        }
        const [branches, tags, analyses] = await Promise.all([
            lastValueFrom(this._projectService.getVCSRepositoryBranches(key, vcsName, repoName, 50)),
            lastValueFrom(this._projectService.getVCSRepositoryTags(key, vcsName, repoName)),
            lastValueFrom(this._projectService.listVCSRepositoryAnalysis(key, vcsName, repoName)),
            events
        ]);
        if (loadId !== this._loadId) {
            return;
        }
        this.branches$.next(branches ?? []);
        this.tags$.next(tags ?? []);
        this.analyses$.next(analyses ?? []);
        this.ref$.next(this.resolveRef(refFromUrl));
        await this.readEntities(loadId);
    }

    /** Applies the ref the url carries, falling back to the remembered one, then to the default branch. */
    async selectRef(refFromUrl: string): Promise<void> {
        if (!this.repository$.value || this.repository$.value.distant) {
            return;
        }
        const ref = this.resolveRef(refFromUrl);
        if (ref === this.ref$.value) {
            return;
        }
        this.ref$.next(ref);
        this.entities$.next(null);
        await this.readEntities(this._loadId);
    }

    /** How many entities of a type the current ref has; null until they are read. */
    entityCount(type: EntityType): number {
        const entities = this.entities$.value;
        return entities ? entities.get(type)?.length ?? 0 : null;
    }

    reloadEntities(): Promise<void> {
        return this.readEntities(this._loadId);
    }

    /** The entities of the current ref; a ref without any leaves an empty map, not null. */
    private async readEntities(loadId: number): Promise<void> {
        const ref = this.ref$.value;
        const entities = await lastValueFrom(this._projectService.getRepoEntities(this._project.key, this._vcsName, this._repoName, ref));
        if (loadId === this._loadId && ref === this.ref$.value) {
            this.entities$.next(groupEntities(entities));
        }
    }

    /** Remembers the ref the user picked; the default branch is the implicit choice and is not stored. */
    rememberRef(ref: string): void {
        const state = this._store.selectSnapshot(PreferencesState.selectProjectRefSelectState(this._project.key));
        if (ref === this.defaultRef()) {
            delete state[this.repositoryPath];
        } else {
            state[this.repositoryPath] = ref;
        }
        this._store.dispatch(new actionPreferences.SaveProjectRefSelectState({ projectKey: this._project.key, state }));
    }

    reloadEvents(): Promise<void> {
        return this.readEvents(this._loadId);
    }

    /**
     * Events failing to read do not fail the page: for a listened repository, it usually means the
     * vcs credentials of the project cannot reach it, which the page explains.
     */
    private async readEvents(loadId: number): Promise<void> {
        try {
            const events = await lastValueFrom(this._projectService.listRepositoryEvents(this._project.key, this._vcsName, this._repoName));
            if (loadId === this._loadId) {
                this.events$.next(events ?? []);
                this.eventsError$.next(null);
            }
        } catch (e) {
            if (loadId === this._loadId) {
                this.events$.next([]);
                this.eventsError$.next(e);
            }
        }
    }

    async reloadAnalyses(): Promise<void> {
        const loadId = this._loadId;
        const analyses = await lastValueFrom(this._projectService.listVCSRepositoryAnalysis(this._project.key, this._vcsName, this._repoName));
        if (loadId === this._loadId) {
            this.analyses$.next(analyses ?? []);
        }
    }

    /** The most recent analysis of a ref, whatever its status. */
    lastAnalysis(ref: string): RepositoryAnalysis {
        return (this.analyses$.value ?? [])
            .filter(a => a.ref === ref)
            .sort((a, b) => Date.parse(b.created) - Date.parse(a.created))[0] ?? null;
    }

    defaultRef(): string {
        const branches = this.branches$.value;
        const branch = branches.find(b => b.default) ?? branches[0];
        return branch ? 'refs/heads/' + branch.display_id : null;
    }

    /** The url wins if it names a ref that exists, then the remembered one, then the default branch. */
    private resolveRef(refFromUrl: string): string {
        if (this.refExists(refFromUrl)) {
            return refFromUrl;
        }
        const remembered = this._store.selectSnapshot(PreferencesState.selectProjectRefSelectState(this._project.key))[this.repositoryPath];
        if (this.refExists(remembered)) {
            return remembered;
        }
        return this.defaultRef();
    }

    private refExists(ref: string): boolean {
        return !!ref && (
            this.branches$.value.some(b => 'refs/heads/' + b.display_id === ref)
            || this.tags$.value.some(t => 'refs/tags/' + t.tag === ref)
        );
    }
}
