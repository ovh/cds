import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { HookEventWorkflowStatus, Project, ProjectRepository, RepositoryHookEvent } from 'app/model/project.model';
import { ProjectState } from 'app/store/project.state';
import { finalize } from 'rxjs/operators';
import { Store } from '@ngxs/store';
import { forkJoin, Subscription } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { VCSProject } from 'app/model/vcs.model';
import { RepositoryAnalysis } from 'app/model/analysis.model';

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
    selectedAnalysisEntities: { [key: string]: { success: { nb: number, files: string[] }, skipped: { nb: number, files: string[] } } }

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _projectService: ProjectService,
        private _cd: ChangeDetectorRef,
        private _toastService: ToastService,
        private _router: Router
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
                this.loadHookEvents();
                this._cd.markForCheck();
            });
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    loadHookEvents(): void {
        this.loadingHooks = true;
        this._cd.markForCheck();
        this._projectService.listRepositoryEvents(this.project.key, this.vcsProject.name, this.repository.name)
            .pipe(finalize(() => {
                this.loadingHooks = false;
                this._cd.markForCheck();
            }))
            .subscribe(hooks => {
                this.hookEvents = hooks.reverse();
                if (this.hookEvents) {
                    this.hookEvents.forEach(he => {
                        if (he.workflows) {
                            let workflowInProject = he.workflows.filter(w => w.project_key === this.project.key);
                            he.nbFailed = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Error).length;
                            he.nbDone = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Done).length;
                            he.nbSkipped = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Skipped).length;
                            he.nbScheduled = workflowInProject.filter(w => w.status === HookEventWorkflowStatus.Scheduled).length;
                        }
                    })
                }
                this._cd.markForCheck();
            })
    }

    removeRepositoryFromProject(): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.deleteVCSRepository(this.project.key, this.vcsProject.name, this.repository.name)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toastService.success('Repository has been removed', '');
                this._router.navigate(['/', 'project', this.project.key]);
            });
    }

    displayDetail(h: RepositoryHookEvent): void {
        let a = h?.analyses?.filter(a => a.project_key === this.project.key);
        if (a?.length === 1) {
            this._projectService.getAnalysis(this.project.key, h.vcs_server_name, h.repository_name, a[0].analyze_id)
                .pipe(finalize(() => {
                    this.selectedHookEvent = h;
                    this._cd.markForCheck();
                }))
                .subscribe(a => {
                    this.selectedAnalysis = <RepositoryAnalysis>a;
                    this.selectedAnalysisEntities = {};
                    if (this.selectedAnalysis.data?.entities) {
                        this.selectedAnalysis.data?.entities.forEach(e => {
                            let type = e.path.replace('.cds/', '').replace('/', '');
                            type = type.charAt(0).toUpperCase() + type.slice(1);
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
                    }
                });

        } else {
            this.selectedHookEvent = h;
            this._cd.markForCheck();
        }
    }

    closeModal(): void {
        delete this.selectedHookEvent;
        delete this.selectedAnalysis;
        delete this.selectedAnalysisEntities;
    }
}
