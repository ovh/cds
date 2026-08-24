import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    inject,
    Input,
    OnDestroy,
    OnInit
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectService } from 'app/service/project/project.service';
import { Project, ProjectRepository } from 'app/model/project.model';
import { filter, lastValueFrom, Subscription } from 'rxjs';
import { AnalysisService } from "app/service/analysis/analysis.service";
import { Entity, EntityType, EntityTypeUtil } from "app/model/entity.model";
import { VCSProject } from 'app/model/vcs.model';
import { NzMessageService } from 'ng-zorro-antd/message';
import { Branch, Tag } from 'app/model/repositories.model';
import { Store } from '@ngxs/store';
import { PreferencesState } from 'app/store/preferences.state';
import * as actionPreferences from 'app/store/preferences.action';
import { ActivatedRoute, NavigationEnd, Router } from '@angular/router';
import { RouterService } from 'app/service/services.module';
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from '../run-start/run-start.component';
import { NzDrawerService } from 'ng-zorro-antd/drawer';
import { ErrorUtils } from 'app/shared/error.utils';
import { ProjectV2RepositoryAddComponent, ProjectV2RepositoryAddComponentParams } from './repository-add/repository-add.component';
import { EventV2State } from 'app/store/event-v2.state';
import { EventV2Type, FullEventV2 } from 'app/model/event-v2.model';
import { AuthenticationState } from 'app/store/authentication.state';
import { ProjectV2TriggerAnalysisComponent, ProjectV2TriggerAnalysisComponentParams } from './trigger-analysis/trigger-analysis.component';

@Component({
    standalone: false,
    selector: 'app-projectv2-explore-sidebar',
    templateUrl: './explore-sidebar.html',
    styleUrls: ['./explore-sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ExploreSidebarComponent implements OnInit, OnDestroy, AfterViewInit {
    /**
     * The tree of the last project explored. The view is rebuilt from scratch every time it is opened,
     * and reading a repository goes through the vcs provider: without this, coming back to it means
     * waiting for the whole tree again. Kept on the class, since it must outlive the component, and
     * for one project only, the one being explored.
     */
    private static snapshot: {
        projectKey: string;
        vcss: Array<VCSProject>;
        repositories: { [vcs: string]: Array<ProjectRepository> };
        entities: { [repositoryPath: string]: Map<EntityType, Array<Entity>> };
        branches: { [repositoryPath: string]: Array<Branch> };
        tags: { [repositoryPath: string]: Array<Tag> };
    };

    @Input() project: Project;

    loading: boolean = true;
    loadingEntities: { [repositoryPath: string]: boolean } = {};
    vcss: Array<VCSProject> = [];
    repositories: { [vcs: string]: Array<ProjectRepository> } = {};
    entities: { [repositoryPath: string]: Map<EntityType, Array<Entity>> } = {};
    treeExpandState: { [key: string]: boolean } = {};
    branches: { [repositoryPath: string]: Array<Branch> } = {};
    tags: { [repositoryPath: string]: Array<Tag> } = {};
    refSelectState: { [repositoryPath: string]: string } = {};
    analysisServiceSub: Subscription;
    routerSub: Subscription;
    eventV2Subscription: Subscription;

    private _cd = inject(ChangeDetectorRef);
    private _projectService = inject(ProjectService);
    private _analysisService = inject(AnalysisService);
    private _messageService = inject(NzMessageService);
    private _store = inject(Store);
    private _router = inject(Router);
    private _routerService = inject(RouterService);
    private _activatedRoute = inject(ActivatedRoute);
    private _drawerService = inject(NzDrawerService);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.treeExpandState = this._store.selectSnapshot(PreferencesState.selectProjectTreeExpandState(this.project.key));
        this.refSelectState = this._store.selectSnapshot(PreferencesState.selectProjectRefSelectState(this.project.key));

        // Coming back to the explore view draws the tree it was left with at once, then reads it
        // again in the background rather than showing nothing until every repository has answered.
        const snapshot = ProjectV2ExploreSidebarComponent.snapshot;
        if (snapshot && snapshot.projectKey === this.project.key) {
            this.vcss = snapshot.vcss;
            this.repositories = snapshot.repositories;
            this.entities = snapshot.entities;
            this.branches = snapshot.branches;
            this.tags = snapshot.tags;
        }

        this.load();

        this.routerSub = this._router.events.pipe(
            filter(e => e instanceof NavigationEnd),
        ).subscribe(() => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            this.expandTreeToSelectedRoute(params).then(() => {
                this._cd.markForCheck();
            });
        });

        this.eventV2Subscription = this._store.select(EventV2State.last).subscribe((event) => {
            this.handleEvent(event);
        });
    }

    ngAfterViewInit(): void {
        this.analysisServiceSub = this._analysisService.getObservable().subscribe(e => { });
    }

    async load() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            this.vcss = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
            this.vcss.map(vcs => {
                if (Object.keys(this.treeExpandState).indexOf(vcs.name) === -1) {
                    this.treeExpandState[vcs.name] = true;
                }
            });
            this._cd.markForCheck();
            await this.loadRepositories();
            ProjectV2ExploreSidebarComponent.snapshot = {
                projectKey: this.project.key,
                vcss: this.vcss,
                repositories: this.repositories,
                entities: this.entities,
                branches: this.branches,
                tags: this.tags
            };
        } catch (e: any) {
            this._messageService.error(`Unable to load vcs and repositories: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        this.loading = false;
        this._cd.markForCheck();
    }

    async loadRepositories() {
        const resp = await Promise.all(this.vcss.map(vcs => lastValueFrom(this._projectService.getVCSRepositories(this.project.key, vcs.name))));
        this.repositories = {};
        this.vcss.forEach((vcs, i) => {
            this.repositories[vcs.name] = resp[i];
        });
        // The repositories are shown as soon as they are known, without waiting for their content.
        this._cd.markForCheck();

        // What the url points at is expanded before reading anything, so that its repository joins
        // the batch instead of being read once every other one has answered.
        this.expandToRoute(this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root));

        await Promise.all(this.vcss.flatMap(vcs => (this.repositories[vcs.name] ?? [])
            .filter(repo => this.treeExpandState[vcs.name + '/' + repo.name])
            .map(repo => this.loadRepository(vcs, repo))));
    }

    async loadRepository(vcs: VCSProject, repo: ProjectRepository) {
        const key = vcs.name + '/' + repo.name;

        // Nothing to show for that repository yet: without this, the tree announces "No resource
        // found" until its entities answer. A repository that already shows some is refreshed
        // silently, keeping them.
        if (!this.entities[key]) {
            this.loadingEntities[key] = true;
            this._cd.markForCheck();
        }

        try {
            // The entities of a ref only need that ref. With one already selected, they are read at
            // the same time as the branches and the tags of the repository, which come from the vcs
            // provider and are the slowest of the three, instead of after them.
            const selectedRef = this.refSelectState[key];
            const entitiesOfSelectedRef = selectedRef
                ? lastValueFrom(this._projectService.getRepoEntities(this.project.key, vcs.name, repo.name, selectedRef)).catch(() => null)
                : null;

            const [branches, tags] = await Promise.all([
                lastValueFrom(this._projectService.getVCSRepositoryBranches(this.project.key, vcs.name, repo.name, 50)),
                lastValueFrom(this._projectService.getVCSRepositoryTags(this.project.key, vcs.name, repo.name))
            ]);
            this.branches[key] = branches;
            this.tags[key] = tags;

            const refExists = !!selectedRef && (
                !!branches.find(b => 'refs/heads/' + b.display_id === selectedRef)
                || !!tags.find(t => 'refs/tags/' + t.tag === selectedRef)
            );
            if (!refExists) {
                this.refSelectState[key] = 'refs/heads/' + branches.find(b => b.default).display_id;
            }

            // The ref that was selected is gone: what was read for it is dropped and the default
            // branch is read instead.
            const entities = (refExists ? await entitiesOfSelectedRef : null)
                ?? await lastValueFrom(this._projectService.getRepoEntities(this.project.key, vcs.name, repo.name, this.refSelectState[key]));
            this.applyEntities(vcs, repo, entities);
        } catch (e: any) {
            this._messageService.error(`Unable to load repository: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        // Each repository shows up as it answers, rather than all of them once the slowest has.
        this.loadingEntities[key] = false;
        this._cd.markForCheck();
    }

    clickVCS(vcs: VCSProject): void {
        this.treeExpandState[vcs.name] = !this.treeExpandState[vcs.name];
        this.saveTreeExpandState();
        this._cd.markForCheck();
    }

    async clickRepository(vcs: VCSProject, repo: ProjectRepository) {
        this.treeExpandState[vcs.name + '/' + repo.name] = !this.treeExpandState[vcs.name + '/' + repo.name];
        this.saveTreeExpandState();

        if (this.treeExpandState[vcs.name + '/' + repo.name]) {
            this.loadingEntities[vcs.name + '/' + repo.name] = true;
            this._cd.markForCheck();
            await this.loadRepository(vcs, repo);
        }

        this.loadingEntities[vcs.name + '/' + repo.name] = false;
        this._cd.markForCheck();
    }

    async clickRepositoryLink(vcs: VCSProject, repo: ProjectRepository, e: Event) {
        e.stopPropagation();

        if (!this.treeExpandState[vcs.name + '/' + repo.name]) {
            this.treeExpandState[vcs.name + '/' + repo.name] = true;
            this.saveTreeExpandState();

            if (this.treeExpandState[vcs.name + '/' + repo.name]) {
                this.loadingEntities[vcs.name + '/' + repo.name] = true;
                this._cd.markForCheck();
                await this.loadRepository(vcs, repo);
            }

            this.loadingEntities[vcs.name + '/' + repo.name] = false;
            this._cd.markForCheck();
        }
    }

    async loadEntities(vcs: VCSProject, repo: ProjectRepository) {
        const resp = await lastValueFrom(this._projectService.getRepoEntities(this.project.key, vcs.name, repo.name, this.refSelectState[vcs.name + '/' + repo.name]));
        this.applyEntities(vcs, repo, resp);
    }

    private applyEntities(vcs: VCSProject, repo: ProjectRepository, resp: Array<Entity>) {
        if (resp.length === 0) {
            this.entities[vcs.name + '/' + repo.name] = null;
            return
        }
        let m = new Map<EntityType, Array<Entity>>();
        resp.forEach(entity => {
            if (!m.has(entity.type)) { m.set(entity.type, []); }
            let entities = m.get(entity.type);
            entities.push(entity);
            m.set(entity.type, entities);
        });
        m.forEach((entities, type) => {
            entities.sort((a, b) => a.name < b.name ? -1 : 1);
            const typeKey = vcs.name + '/' + repo.name + '/' + type;
            // Workflows are the usual entry point of a repository; opening every
            // folder buries them, so the others stay closed until asked for.
            if (Object.keys(this.treeExpandState).indexOf(typeKey) === -1) {
                this.treeExpandState[typeKey] = type === EntityType.Workflow;
            }
        });
        this.entities[vcs.name + '/' + repo.name] = m;
    }

    async clickEntityType(vcs: VCSProject, repo: ProjectRepository, type: EntityType) {
        this.treeExpandState[vcs.name + '/' + repo.name + '/' + type] = !this.treeExpandState[vcs.name + '/' + repo.name + '/' + type];
        this.saveTreeExpandState();
        this._cd.markForCheck();
    }

    collapseAll(): void {
        // Check if any repository or entity type is expanded
        const hasExpandedChildren = Object.keys(this.treeExpandState).some(key => {
            return key.includes('/') && this.treeExpandState[key];
        });

        if (hasExpandedChildren) {
            // First click: collapse all repositories and entity types
            Object.keys(this.treeExpandState).forEach(key => {
                if (key.includes('/')) {
                    this.treeExpandState[key] = false;
                }
            });
        } else {
            // Second click: collapse all VCS
            Object.keys(this.treeExpandState).forEach(key => {
                this.treeExpandState[key] = false;
            });
        }

        this.saveTreeExpandState();

        // Navigate to explore overview if we're currently viewing an entity
        const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
        if (params['vcsName'] || params['repoName'] || params['entityType'] || params['entityName']) {
            this._router.navigate(['/project', this.project.key, 'explore']);
        }

        this._cd.markForCheck();
    }

    async selectRepositoryRef(vcs: VCSProject, repo: ProjectRepository, ref: string) {
        this.refSelectState[vcs.name + '/' + repo.name] = ref;
        this.saveRefSelectState();

        try {
            await this.loadEntities(vcs, repo);
        } catch (e: any) {
            this._messageService.error(`Unable to load repository: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
        if (params['vcsName'] === vcs.name && params['repoName'] === repo.name) {
            if (params['entityType'] && params['entityName']) {
                let entityType = EntityTypeUtil.fromURLParam(params['entityType']);
                let entityName = params['entityName'];
                this._router.navigate(['/project', this.project.key, 'explore', 'vcs', vcs.name, 'repository', repo.name, EntityTypeUtil.toURLParam(entityType), entityName], {
                    queryParams: {
                        ref: this.refSelectState[vcs.name + '/' + repo.name]
                    }
                });
            }
        }

        this._cd.markForCheck();
    }

    saveTreeExpandState(): void {
        let state: { [key: string]: boolean } = {};
        const keys = Object.keys(this.treeExpandState);
        // Persist vcs that were closed by user
        this.vcss.forEach(vcs => {
            if (keys.indexOf(vcs.name) !== -1 && !this.treeExpandState[vcs.name]) {
                state[vcs.name] = false;
            }
            // Persist repositories that were opened or closed
            (this.repositories[vcs.name] ?? []).forEach(repo => {
                const repoKey = vcs.name + '/' + repo.name;
                if (keys.indexOf(repoKey) !== -1) {
                    state[repoKey] = this.treeExpandState[repoKey];
                }
                // Persist entity folders that were opened or closed. Both directions
                // matter since the default is no longer "everything open".
                (this.entities[repoKey] ?? new Map<EntityType, Array<Entity>>()).forEach((_, entityType) => {
                    const typeKey = repoKey + '/' + entityType;
                    if (keys.indexOf(typeKey) !== -1) {
                        state[typeKey] = this.treeExpandState[typeKey];
                    }
                });
            });
        });
        this._store.dispatch(new actionPreferences.SaveProjectTreeExpandState({ projectKey: this.project.key, state }));
    }

    async expandTreeToSelectedRoute(params: any) {
        const target = this.expandToRoute(params);
        if (target?.read) {
            await this.loadRepository(target.vcs, target.repo);
        }
    }

    /**
     * Expands the branch of the tree the url points at, reading nothing. Tells whether the content of
     * the repository it names has to be read, so that the caller can decide when.
     */
    private expandToRoute(params: any): { vcs: VCSProject, repo: ProjectRepository, read: boolean } {
        const vcs = this.vcss.find(vcs => vcs.name === params['vcsName']);
        if (!vcs) {
            return null;
        }
        this.treeExpandState[vcs.name] = true;

        const repo = (this.repositories[vcs.name] ?? []).find(repo => repo.name === params['repoName']);
        if (!repo) {
            return null;
        }
        const key = vcs.name + '/' + repo.name;

        let read = false;
        if (!this.treeExpandState[key]) {
            this.treeExpandState[key] = true;
            read = true;
        }
        const ref = this._activatedRoute.snapshot.queryParams['ref'];
        if (ref && this.refSelectState[key] !== ref) {
            this.refSelectState[key] = ref;
            read = true;
        }

        if (params['entityType']) {
            try {
                this.treeExpandState[key + '/' + EntityTypeUtil.fromURLParam(params['entityType'])] = true;
            } catch (e) {
                // URL carries an unknown entity type, nothing to expand
            }
        }

        return { vcs, repo, read };
    }

    saveRefSelectState(): void {
        let state: { [key: string]: string } = {};
        this.vcss.forEach(vcs => {
            // Persist selected ref only when different from default branch
            (this.repositories[vcs.name] ?? []).forEach(repo => {
                if (!this.branches[vcs.name + '/' + repo.name]) {
                    return;
                }
                const defaultBranch = this.branches[vcs.name + '/' + repo.name].find(b => b.default).display_id;
                if (this.refSelectState[vcs.name + '/' + repo.name] === 'refs/heads/' + defaultBranch) {
                    return;
                }
                state[vcs.name + '/' + repo.name] = this.refSelectState[vcs.name + '/' + repo.name];
            });
        });
        this._store.dispatch(new actionPreferences.SaveProjectRefSelectState({ projectKey: this.project.key, state }));
    }

    openRunStartDrawer(workflow: string, ref: string): void {
        const drawerRef = this._drawerService.create<ProjectV2RunStartComponent, { value: string }, string>({
            nzTitle: 'Start new Workflow Run',
            nzContent: ProjectV2RunStartComponent,
            nzContentParams: {
                params: <ProjectV2RunStartComponentParams>{
                    workflow,
                    workflow_ref: ref
                }
            },
            nzSize: 'large',
            nzBodyStyle: { 'padding': '0' }
        });
        drawerRef.afterClose.subscribe(data => { });
    }

    openRepositoryAddDrawer(vcs: string): void {
        const drawerRef = this._drawerService.create<ProjectV2RepositoryAddComponent, { value: string }, string>({
            nzTitle: 'Add a new Repository',
            nzContent: ProjectV2RepositoryAddComponent,
            nzContentParams: {
                params: <ProjectV2RepositoryAddComponentParams>{
                    vcs
                }
            },
            nzSize: 'large'
        });
        drawerRef.afterClose.subscribe(data => { });
    }

    openTriggerAnalysisDrawer(repository: string, ref: string): void {
        const drawerRef = this._drawerService.create<ProjectV2TriggerAnalysisComponent, { value: string }, string>({
            nzTitle: 'Trigger repository analysis',
            nzContent: ProjectV2TriggerAnalysisComponent,
            nzContentParams: {
                params: <ProjectV2TriggerAnalysisComponentParams>{
                    repository,
                    ref
                }
            },
            nzSize: 'large'
        });
        drawerRef.afterClose.subscribe(data => { });
    }

    async handleEvent(event: FullEventV2) {
        if (!event || [EventV2Type.EventRepositoryCreated, EventV2Type.EventRepositoryDeleted, EventV2Type.EventAnalysisDone].indexOf(event.type) === -1) { return; }

        if (!this.repositories[event.vcs_name]) {
            return;
        }

        if (event.type === EventV2Type.EventRepositoryDeleted) {
            this.repositories[event.vcs_name] = this.repositories[event.vcs_name].filter(r => r.name !== event.repository);
            delete this.treeExpandState[event.vcs_name + '/' + event.repository];
            this.saveTreeExpandState();
            this._cd.markForCheck();
            return
        }

        let repository = this.repositories[event.vcs_name].find(r => r.name === event.repository);
        if (!repository) {
            repository = await lastValueFrom(this._projectService.getVCSRepository(this.project.key, event.vcs_name, event.repository));
            this.repositories[event.vcs_name].push(repository);
            this.repositories[event.vcs_name].sort((a, b) => a.name < b.name ? -1 : 1);
        }

        const expand = event.username === this._store.selectSnapshot(AuthenticationState.summary).user.username;
        this.treeExpandState[event.vcs_name + '/' + event.repository] = expand;
        if (expand) {
            this.loadingEntities[event.vcs_name + '/' + event.repository] = true;
            this._cd.markForCheck();
            await this.loadRepository(this.vcss.find(vcs => vcs.name === event.vcs_name), repository);
            this.loadingEntities[event.vcs_name + '/' + event.repository] = false;
        }

        this.saveTreeExpandState();
        this._cd.markForCheck();
    }

}
