import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { filter, finalize } from 'rxjs/operators';
import cloneDeep from 'lodash-es/cloneDeep';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';
import { Project, ProjectRepository, VCSProject } from 'app/model/project.model';
import { ProjectService } from 'app/service/project/project.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { Repository } from 'app/model/repositories.model';
import { ToastService } from 'app/shared/toast/ToastService';

@Component({
    selector: 'app-projectv2-repository-add',
    templateUrl: './project.repository.add.html',
    styleUrls: ['./project.repository.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryAddComponent implements OnDestroy {

    projectSubscriber: Subscription
    loading: boolean;

    project: Project;
    vcsProject: VCSProject;

    selectedRepo: Repository;
    repositories: Repository[];
    filteredRepos: Repository[];

    constructor(private _routeActivated: ActivatedRoute, private _store: Store, private _cd: ChangeDetectorRef, private _projectService: ProjectService,
        private _repoManagerService: RepoManagerService, private _toastService: ToastService, private _router: Router) {
        // Get project and VCS, subscribe to react in case of project switch
        this.projectSubscriber = this._store.select(ProjectState)
            .pipe(filter((projState: ProjectStateModel) => {
                return projState && projState.project && projState.project.key !== null && !projState.project.externalChange &&
                    this._routeActivated.parent.snapshot.params['key'] === projState.project.key;
            })).subscribe((projState: ProjectStateModel)  => {
                this.project = cloneDeep(projState.project);
                this._cd.markForCheck();
                this._routeActivated.params.subscribe(p => {
                    this._projectService.getVCSProject(this.project.key, p['vcsName']).subscribe(vcsP => {
                        this.vcsProject = vcsP;
                        this._cd.markForCheck();
                        this.listRepositories(false);

                    });
                });
            });
    }

    listRepositories(resync: boolean): void {
        this._repoManagerService.getRepositories(this.project.key, this.vcsProject.name, resync).subscribe(repos => {
            this.repositories = repos;
            this.filteredRepos = repos.slice(0, 100);
            this._cd.markForCheck();
        });
    }

    filterRepo(query: string): void{
        if (!query || query.length < 3) {
            return;
        }
        this.filteredRepos = this.repositories.filter(repo => repo.fullname.toLowerCase().indexOf(query.toLowerCase()) !== -1);
        this._cd.markForCheck();
    }

    trackRepo(idx: number, r: Repository): string { return r.name; }

    addRepositoryOnProject(): void {
        let repo = new ProjectRepository();
        repo.name = this.selectedRepo.fullname;
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.addVCSRepository(this.project.key, this.vcsProject.name, repo).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe( r => {
            this._toastService.success('Repository added', '');
            this._router.navigate(['/', 'projectv2', this.project.key, 'vcs', this.vcsProject.name, 'repository', r.name]).then()
        });
    }

    ngOnDestroy() {}

}
