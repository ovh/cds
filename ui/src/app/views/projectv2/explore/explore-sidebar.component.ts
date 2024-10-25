import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
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

@Component({
    selector: 'app-projectv2-explore-sidebar',
    templateUrl: './explore-sidebar.html',
    styleUrls: ['./explore-sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ExploreSidebarComponent implements OnInit, OnDestroy, AfterViewInit {
    @Input() project: Project;

    loading: boolean = true;
    loadingEntities: { [repositoryPath: string]: boolean } = {};
    vcss: Array<VCSProject> = [];
    repositories: { [vcs: string]: Array<ProjectRepository> } = {};
    entities: { [repositoryPath: string]: { [entityType: string]: Array<Entity> } } = {};
    treeExpandState: { [key: string]: boolean } = {};
    branches: { [repositoryPath: string]: Array<Branch> } = {};
    tags: { [repositoryPath: string]: Array<Tag> } = {};
    refSelectState: { [repositoryPath: string]: string } = {};
    analysisServiceSub: Subscription;
    routerSub: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _analysisService: AnalysisService,
        private _messageService: NzMessageService,
        private _store: Store,
        private _router: Router,
        private _routerService: RouterService,
        private _activatedRoute: ActivatedRoute,
        private _drawerService: NzDrawerService
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.treeExpandState = this._store.selectSnapshot(PreferencesState.selectProjectTreeExpandState(this.project.key));
        this.refSelectState = this._store.selectSnapshot(PreferencesState.selectProjectRefSelectState(this.project.key));
        this.load();
        this.routerSub = this._router.events.pipe(
            filter(e => e instanceof NavigationEnd),
        ).subscribe(() => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            this.expandTreeToSelectedRoute(params).then(() => {
                this._cd.markForCheck();
            });
        });
    }

    ngAfterViewInit(): void {
        this.analysisServiceSub = this._analysisService.getObservable().subscribe(e => {

        });
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
            await this.loadRepositories();
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            await this.expandTreeToSelectedRoute(params);
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
        // Async load each repository expanded
        let promises = [];
        this.vcss.forEach((vcs, i) => {
            this.repositories[vcs.name].forEach(repo => {
                if (this.treeExpandState[vcs.name + '/' + repo.name]) {
                    promises.push(this.loadRepository(vcs, repo));
                }
            });
        });
        await Promise.all(promises);
    }

    async loadRepository(vcs: VCSProject, repo: ProjectRepository) {
        try {
            const branches = await lastValueFrom(this._projectService.getVCSRepositoryBranches(this.project.key, vcs.name, repo.name, 50));
            this.branches[vcs.name + '/' + repo.name] = branches;
            const tags = await lastValueFrom(this._projectService.getVCSRepositoryTags(this.project.key, vcs.name, repo.name));
            this.tags[vcs.name + '/' + repo.name] = tags;
            if (!this.refSelectState[vcs.name + '/' + repo.name] || (
                !branches.find(b => 'refs/heads/' + b.display_id === this.refSelectState[vcs.name + '/' + repo.name])
                && !tags.find(t => 'refs/tags/' + t.tag === this.refSelectState[vcs.name + '/' + repo.name])
            )) {
                this.refSelectState[vcs.name + '/' + repo.name] = 'refs/heads/' + branches.find(b => b.default).display_id;
            }
            await this.loadEntities(vcs, repo);
        } catch (e: any) {
            this._messageService.error(`Unable to load repository: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
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
        if (resp.length === 0) {
            this.entities[vcs.name + '/' + repo.name] = null;
            return
        }
        let m = {};
        resp.forEach(entity => {
            if (!m[entity.type]) { m[entity.type] = []; }
            m[entity.type].push(entity);
        });
        Object.keys(m).forEach(key => {
            m[key].sort((a, b) => { a.name < b.name ? -1 : 1 });
            if (Object.keys(this.treeExpandState).indexOf(vcs.name + '/' + repo.name + '/' + key) === -1) {
                this.treeExpandState[vcs.name + '/' + repo.name + '/' + key] = true;
            }
        });
        this.entities[vcs.name + '/' + repo.name] = m;
    }

    async clickEntityType(vcs: VCSProject, repo: ProjectRepository, type: EntityType) {
        this.treeExpandState[vcs.name + '/' + repo.name + '/' + type] = !this.treeExpandState[vcs.name + '/' + repo.name + '/' + type];
        this.saveTreeExpandState();
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

    clickRefresh() {
        this.load();
    }

    saveTreeExpandState(): void {
        let state: { [key: string]: boolean } = {};
        const keys = Object.keys(this.treeExpandState);
        // Persist vcs that were closed by user
        this.vcss.forEach(vcs => {
            if (keys.indexOf(vcs.name) !== -1 && !this.treeExpandState[vcs.name]) {
                state[vcs.name] = false;
            }
            // Persist repositories that were opened
            (this.repositories[vcs.name] ?? []).forEach(repo => {
                if (this.treeExpandState[vcs.name + '/' + repo.name]) {
                    state[vcs.name + '/' + repo.name] = true;
                }
                // Persist entity folder that were closed
                Object.keys(this.entities[vcs.name + '/' + repo.name] ?? {}).forEach(entityType => {
                    if (keys.indexOf(vcs.name + '/' + repo.name + '/' + entityType) !== -1 && !this.treeExpandState[vcs.name + '/' + repo.name + '/' + entityType]) {
                        state[vcs.name + '/' + repo.name + '/' + entityType] = false;
                    }
                });
            });
        });
        this._store.dispatch(new actionPreferences.SaveProjectTreeExpandState({ projectKey: this.project.key, state }));
    }

    async expandTreeToSelectedRoute(params: any) {
        if (!params['vcsName'] || this.vcss.findIndex(vcs => vcs.name === params['vcsName']) < 0) {
            return;
        }
        const vcs = this.vcss.find(vcs => vcs.name === params['vcsName']);
        this.treeExpandState[vcs.name] = true;
        if (!params['repoName'] || this.repositories[vcs.name].findIndex(repo => repo.name === params['repoName']) < 0) {
            return;
        }
        let loadRepo = false;
        const repo = this.repositories[vcs.name].find(repo => repo.name === params['repoName'])
        if (!this.treeExpandState[vcs.name + '/' + repo.name]) {
            this.treeExpandState[vcs.name + '/' + repo.name] = true;
            loadRepo = true;
        }
        const ref = this._activatedRoute.snapshot.queryParams['ref'];
        if (ref && this.refSelectState[vcs.name + '/' + repo.name] !== ref) {
            this.refSelectState[vcs.name + '/' + repo.name] = ref;
            loadRepo = true;
        }
        if (loadRepo) {
            await this.loadRepository(vcs, repo);
        }
        let entityType: EntityType = null;
        if (params['workflowName']) {
            entityType = EntityType.Workflow;
        } else if (params['actionName']) {
            entityType = EntityType.Action;
        } else if (params['workerModelName']) {
            entityType = EntityType.WorkerModel;
        }
        if (entityType) {
            this.treeExpandState[vcs.name + '/' + repo.name + '/' + entityType] = true;
        }
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
            nzTitle: 'Start new worklfow run',
            nzContent: ProjectV2RunStartComponent,
            nzContentParams: {
                params: <ProjectV2RunStartComponentParams>{
                    workflow,
                    workflow_ref: ref
                }
            },
            nzSize: 'large'
        });
        drawerRef.afterClose.subscribe(data => { });
    }

}