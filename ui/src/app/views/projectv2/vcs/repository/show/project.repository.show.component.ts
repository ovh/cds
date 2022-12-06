import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Project, ProjectRepository, VCSProject } from 'app/model/project.model';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { filter, finalize } from 'rxjs/operators';
import cloneDeep from 'lodash-es/cloneDeep';
import { Store } from '@ngxs/store';
import { forkJoin, Observable, Subscription } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { SidebarEvent, SidebarService } from 'app/service/sidebar/sidebar.service';
import { FlatNodeItem } from 'app/shared/tree/tree.component';

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
                private _cd: ChangeDetectorRef, private _toastService: ToastService, private _router: Router, private _sidebarService: SidebarService) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this._routeActivated.params.subscribe(p => {
            if (this.vcsProject?.name === p['vcsName'] && this.repository?.name === p['repoName']) {
                return;
            }

            forkJoin( [
                this._projectService.getVCSRepository(this.project.key, p['vcsName'], p['repoName']),
                this._projectService.getVCSProject(this.project.key, p['vcsName'])
            ]).subscribe(result => {
                this.repository = result[0];
                this.vcsProject = result[1];
                let selectEvent = new SidebarEvent(this.repository.id, this.repository.name, 'repository', 'select', <FlatNodeItem>{id: this.vcsProject.id, name: this.vcsProject.name, type: 'vcs'});
                this._sidebarService.sendEvent(selectEvent);
                this._cd.markForCheck();
            })
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
