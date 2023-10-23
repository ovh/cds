import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {Project, ProjectRepository, RepositoryHookEvent, VCSProject} from 'app/model/project.model';
import { ProjectState } from 'app/store/project.state';
import { finalize } from 'rxjs/operators';
import { Store } from '@ngxs/store';
import { forkJoin, Subscription } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { SidebarEvent, SidebarService } from 'app/service/sidebar/sidebar.service';
import { RepositoryAnalysis } from "../../../../../model/analysis.model";
import { AnalysisService } from "../../../../../service/analysis/analysis.service";

@Component({
    selector: 'app-projectv2-repository-show',
    templateUrl: './project.repository.show.html',
    styleUrls: ['./project.repository.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryShowComponent implements OnDestroy, OnInit {

    loading: boolean;
    loadingAnalysis: boolean;
    loadingHooks : boolean;
    projectSubscriber: Subscription;

    project: Project;
    vcsProject: VCSProject;
    repository: ProjectRepository;
    repoAnalyses: Array<RepositoryAnalysis>;
    hookEvents: Array<RepositoryHookEvent>;

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _projectService: ProjectService,
        private _analyzeService: AnalysisService,
        private _cd: ChangeDetectorRef,
        private _toastService: ToastService,
        private _router: Router,
        private _sidebarService: SidebarService
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
                let selectEvent = new SidebarEvent(this.repository.id, this.repository.name, 'repository', 'select', [this.vcsProject.id]);
                this._sidebarService.sendEvent(selectEvent);
                this.loadAnalyses();
                this.loadHookEvents();
                this._cd.markForCheck();
            });
        });
    }

    ngOnInit() {
        this._analyzeService.getObservable().subscribe(e => {
            if (e && this.repository && this.vcsProject) {
                this.loadAnalyses();
            }
        });
    }

    loadHookEvents(): void {
        this.loadingHooks = true;
        this._cd.markForCheck();
        this._projectService.loadRepositoryHooks(this.project.key, this.vcsProject.name, this.repository.name)
            .pipe(finalize(() => {
                this.loadingHooks = false;
                this._cd.markForCheck();
            }))
            .subscribe(hooks => {
                this.hookEvents = hooks.reverse();
                this._cd.markForCheck();
            })
    }

    loadAnalyses(): void {
        this.loadingAnalysis = true;
        this._cd.markForCheck();
        this._projectService.listVCSRepositoryAnalysis(this.project.key, this.vcsProject.name, this.repository.name)
            .pipe(finalize(() => {
                this.loadingAnalysis = false;
                this._cd.markForCheck();
            }))
            .subscribe(analyses => {
                this.repoAnalyses = analyses.reverse();
                this._cd.markForCheck();
            });
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
                this._router.navigate(['/', 'projectv2', this.project.key]);
            });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT
}
