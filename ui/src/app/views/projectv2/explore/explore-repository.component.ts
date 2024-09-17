import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { HookEventWorkflowStatus, Project, ProjectRepository, RepositoryHookEvent, WorkflowHookEventName } from 'app/model/project.model';
import { ProjectState } from 'app/store/project.state';
import { Store } from '@ngxs/store';
import { forkJoin, lastValueFrom, Subscription } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { VCSProject } from 'app/model/vcs.model';
import { RepositoryAnalysis } from 'app/model/analysis.model';
import { NzDrawerService } from 'ng-zorro-antd/drawer';
import { ProjectV2TriggerAnalysisComponent } from './trigger-analysis/trigger-analysis.component';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzTableFilterList } from 'ng-zorro-antd/table';

@Component({
    selector: 'app-projectv2-explore-repository',
    templateUrl: './explore-repository.html',
    styleUrls: ['./explore-repository.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ExploreRepositoryComponent implements OnDestroy {
    loading: boolean;
    loadingHooks: boolean;
    projectSubscriber: Subscription;
    project: Project;
    vcsProject: VCSProject;
    repository: ProjectRepository;
    hookEvents: Array<RepositoryHookEvent>;
    selectedHookEvent: RepositoryHookEvent
    selectedAnalysis: RepositoryAnalysis;
    selectedAnalysisEntities: { [key: string]: { success: { nb: number, files: string[] }, skipped: { nb: number, files: string[] } } };
    eventFilterList: NzTableFilterList = [
        { text: WorkflowHookEventName.WorkflowHookEventNameManual, value: WorkflowHookEventName.WorkflowHookEventNameManual },
        { text: WorkflowHookEventName.WorkflowHookEventNameModelUpdate, value: WorkflowHookEventName.WorkflowHookEventNameModelUpdate },
        { text: WorkflowHookEventName.WorkflowHookEventNamePullRequest, value: WorkflowHookEventName.WorkflowHookEventNamePullRequest },
        { text: WorkflowHookEventName.WorkflowHookEventNamePullRequestComment, value: WorkflowHookEventName.WorkflowHookEventNamePullRequestComment },
        { text: WorkflowHookEventName.WorkflowHookEventNamePush, value: WorkflowHookEventName.WorkflowHookEventNamePush },
        { text: WorkflowHookEventName.WorkflowHookEventNameScheduler, value: WorkflowHookEventName.WorkflowHookEventNameScheduler },
        { text: WorkflowHookEventName.WorkflowHookEventNameWorkflowUpdate, value: WorkflowHookEventName.WorkflowHookEventNameWorkflowUpdate }
    ];

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _projectService: ProjectService,
        private _cd: ChangeDetectorRef,
        private _toastService: ToastService,
        private _router: Router,
        private _drawerService: NzDrawerService,
        private _messageService: NzMessageService
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this._routeActivated.params.subscribe(p => {
            if (this.vcsProject?.name === p['vcsName'] && this.repository?.name === p['repoName']) {
                return;
            }
            forkJoin([
                this._projectService.getVCSRepository(this.project.key, p['vcsName'], p['repoName']),
                this._projectService.getVCSProject(this.project.key, p['vcsName'])
            ]).subscribe(result => {
                this.repository = result[0];
                this.vcsProject = result[1];
                this._cd.markForCheck();
                this.loadHookEvents();
            });
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    async loadHookEvents() {
        this.loadingHooks = true;
        this._cd.markForCheck();

        this.hookEvents = await lastValueFrom(this._projectService.listRepositoryEvents(this.project.key, this.vcsProject.name, this.repository.name));
        (this.hookEvents ?? []).forEach(he => {
            he.created_string = new Date(he.created / 1000000).toJSON();
            if (he.workflows) {
                const workflowInProject = he.workflows.filter(w => w.project_key === this.project.key);
                he.nbFailed = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Error).length;
                he.nbDone = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Done).length;
                he.nbSkipped = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Skipped).length;
                he.nbScheduled = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Scheduled).length;
            }
        });

        this.loadingHooks = false;
        this._cd.markForCheck();
    }

    async removeRepositoryFromProject() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._projectService.deleteVCSRepository(this.project.key, this.vcsProject.name, this.repository.name));
            this._toastService.success('Repository has been removed', '');
            this._router.navigate(['/', 'project', this.project.key]);
        } catch (e) {
            this._messageService.error(`Unable to remove repository: ${e?.error?.error}`, { nzDuration: 2000 });
        }

        this.loading = false;
        this._cd.markForCheck();
    }

    async displayDetail(h: RepositoryHookEvent) {
        this.selectedHookEvent = h;

        const a = (h?.analyses ?? []).filter(a => a.project_key === this.project.key);
        if (a.length !== 1) {
            this._cd.markForCheck();
            return;
        }

        this.selectedAnalysis = await lastValueFrom(this._projectService.getAnalysis(this.project.key, h.vcs_server_name, h.repository_name, a[0].analyze_id));

        (this.selectedAnalysis.data?.entities ?? []).forEach(e => {
            let type = e.path.replace('.cds/', '').replace('/', '');
            type = type.charAt(0).toUpperCase() + type.slice(1);
            if (!this.selectedAnalysisEntities) { this.selectedAnalysisEntities = {}; }
            if (!this.selectedAnalysisEntities[type]) {
                this.selectedAnalysisEntities[type] = { skipped: { nb: 0, files: [] }, success: { nb: 0, files: [] } };
            }
            if (e.status == 'Success') {
                this.selectedAnalysisEntities[type].success.nb++;
                this.selectedAnalysisEntities[type].success.files.push(e.file_name);
            } else {
                this.selectedAnalysisEntities[type].skipped.nb++;
                this.selectedAnalysisEntities[type].skipped.files.push(e.file_name);
            }
        });

        this._cd.markForCheck();
    }

    closeModal(): void {
        delete this.selectedHookEvent;
        delete this.selectedAnalysis;
        delete this.selectedAnalysisEntities;
        this._cd.markForCheck();
    }

    openTriggerAnalysisDrawer(): void {
        const drawerRef = this._drawerService.create<ProjectV2TriggerAnalysisComponent, { value: string }, string>({
            nzTitle: 'Trigger repository analysis',
            nzContent: ProjectV2TriggerAnalysisComponent,
            nzContentParams: {
                params: {
                    repository: this.repository.name
                }
            },
            nzSize: 'large'
        });
        drawerRef.afterClose.subscribe(data => { });
    }

    sortHookByDate(a: RepositoryHookEvent, b: RepositoryHookEvent): number {
        return a.created < b.created ? -1 : 1;
    }

    eventFilterFunc(eventNames: WorkflowHookEventName[], hookEvent: RepositoryHookEvent): boolean {
        return eventNames.indexOf(hookEvent.event_name) !== -1;
    }
}
