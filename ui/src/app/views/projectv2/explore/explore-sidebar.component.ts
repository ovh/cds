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
import { lastValueFrom, Subscription } from 'rxjs';
import { AnalysisService } from "app/service/analysis/analysis.service";
import { Entity, EntityType } from "app/model/entity.model";
import { VCSProject } from 'app/model/vcs.model';
import { NzMessageService } from 'ng-zorro-antd/message';
import { Branch } from 'app/model/repositories.model';
import { Store } from '@ngxs/store';
import { PreferencesState } from 'app/store/preferences.state';
import * as actionPreferences from 'app/store/preferences.action';

@Component({
    selector: 'app-projectv2-explore-sidebar',
    templateUrl: './explore-sidebar.html',
    styleUrls: ['./explore-sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2SidebarComponent implements OnInit, OnDestroy, AfterViewInit {
    @Input() project: Project;

    loading: boolean = true;
    vcss: Array<VCSProject> = [];
    repositories: { [vcs: string]: Array<ProjectRepository> } = {};
    entities: { [repositoryPath: string]: { [entityType: string]: Array<Entity> } } = {};
    treeExpandState: { [key: string]: boolean } = {};
    branches: { [repositoryPath: string]: Array<Branch> } = {};
    branchSelectState: { [repositoryPath: string]: string } = {};

    analysisServiceSub: Subscription;

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    constructor(
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _analysisService: AnalysisService,
        private _messageService: NzMessageService,
        private _store: Store
    ) { }

    ngOnInit(): void {
        this.treeExpandState = this._store.selectSnapshot(PreferencesState.selectProjectTreeExpandState(this.project.key));
        this.load();
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
        } catch (e: any) {
            this._messageService.error(`Unable to load vcs and repositories: ${e?.error?.error}`, { nzDuration: 2000 });
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
            if (!this.branchSelectState[vcs.name + '/' + repo.name]) {
                this.branchSelectState[vcs.name + '/' + repo.name] = branches.find(b => b.default).display_id;
            }
            await this.loadEntities(vcs, repo);
        } catch (e: any) {
            this._messageService.error(`Unable to load repository: ${e?.error?.error}`, { nzDuration: 2000 });
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
            await this.loadRepository(vcs, repo);
        }

        this._cd.markForCheck();
    }

    clickRepositoryLink(vcs: VCSProject, repo: ProjectRepository, e: Event): void {
        if (this.treeExpandState[vcs.name + '/' + repo.name]) {
            e.stopPropagation();
        }
    }

    async loadEntities(vcs: VCSProject, repo: ProjectRepository) {
        const resp = await lastValueFrom(this._projectService.getRepoEntities(this.project.key, vcs.name, repo.name, this.branchSelectState[vcs.name + '/' + repo.name]));
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

    async selectRepositoryBranch(vcs: VCSProject, repo: ProjectRepository, branch: string) {
        this.branchSelectState[vcs.name + '/' + repo.name] = branch;
        try {
            await this.loadEntities(vcs, repo);
        } catch (e: any) {
            this._messageService.error(`Unable to load repository: ${e?.error?.error}`, { nzDuration: 2000 });
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

}