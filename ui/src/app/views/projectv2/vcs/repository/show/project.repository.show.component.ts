import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Project, ProjectRepository, VCSProject } from 'app/model/project.model';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { filter, finalize } from 'rxjs/operators';
import cloneDeep from 'lodash-es/cloneDeep';
import { Store } from '@ngxs/store';
import { Subscription } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';

@Component({
    selector: 'app-projectv2-repository-show',
    templateUrl: './project.repository.show.html',
    styleUrls: ['./project.repository.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryShowComponent implements OnDestroy {

    loading: boolean;
    projectSubscriber: Subscription;

    project: Project;
    vcsProject: VCSProject;
    repository: ProjectRepository;

    constructor(private _store: Store, private _routeActivated: ActivatedRoute, private _projectService: ProjectService,
                private _cd: ChangeDetectorRef, private _toastService: ToastService, private _router: Router) {
        this.projectSubscriber = this._store.select(ProjectState)
            .pipe(filter((projState: ProjectStateModel) => {
                return projState && projState.project && projState.project.key !== null && !projState.project.externalChange &&
                    this._routeActivated.parent.snapshot.params['key'] === projState.project.key;
            })).subscribe((projState: ProjectStateModel)  => {
                this.project = cloneDeep(projState.project);
                this._cd.markForCheck();
                this._routeActivated.params.subscribe(p => {
                    this._projectService.getVCSRepository(this.project.key, p['vcsName'], p['repoName']).subscribe(repo => {
                        this.repository = repo;
                        this._cd.markForCheck();
                    });
                    this._projectService.getVCSProject(this.project.key, p['vcsName']).subscribe(vcsProject => {
                        this.vcsProject = vcsProject;
                        this._cd.markForCheck();
                    });
                });
            });
    }

    removeRepositoryFromProject(): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.deleteVCSRepository(this.project.key, this.vcsProject.name, this.repository.name)
            .pipe(finalize( () => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toastService.success('Repository has been removed', '');
                this._router.navigate(['/', 'projectv2', this.project.key]);
        })
    }

    ngOnDestroy() {}
}
