import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { ProjectState } from 'app/store/project.state';
import { finalize } from 'rxjs/operators';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { forkJoin } from 'rxjs';
import { Project, ProjectRepository } from 'app/model/project.model';
import { ProjectService } from 'app/service/project/project.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { Repository } from 'app/model/repositories.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { VCSProject } from 'app/model/vcs.model';

@Component({
    selector: 'app-projectv2-repository-add',
    templateUrl: './project.repository.add.html',
    styleUrls: ['./project.repository.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryAddComponent implements OnDestroy {

    loading: boolean;
    loadingResync: boolean;

    project: Project;
    vcsProject: VCSProject;
    currentRepositories: ProjectRepository[];

    selectedRepo: Repository;
    repositories: Repository[];
    filteredRepos: Repository[];

    constructor(private _routeActivated: ActivatedRoute, private _store: Store, private _cd: ChangeDetectorRef, private _projectService: ProjectService,
        private _repoManagerService: RepoManagerService, private _toastService: ToastService, private _router: Router) {
        // Get project and VCS, subscribe to react in case of project switch
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this._routeActivated.params.subscribe(p => {
            forkJoin([
                this._projectService.getVCSRepositories(this.project.key, p['vcsName']),
                this._projectService.getVCSProject(this.project.key, p['vcsName'])
            ]).subscribe(result => {
                this.currentRepositories = result[0];
                this.vcsProject = result[1];
                this._cd.markForCheck();
                this.listRepositories(false);
            });
        });
    }

    listRepositories(resync: boolean): void {
        this.loadingResync = true;
        this._cd.markForCheck();
        this._repoManagerService.getRepositories(this.project.key, this.vcsProject.name, resync)
            .pipe(finalize(() => {
                this.loadingResync = false;
                this._cd.markForCheck();
            }))
            .subscribe(repos => {
                this.repositories = repos.filter(repo => this.currentRepositories.findIndex(r => r.name === repo.fullname) === -1);
                this.filteredRepos = this.repositories.slice(0, 100);
                this._cd.markForCheck();
            });
    }

    filterRepo(query: string): void {
        if (!query || query.length < 3) {
            return;
        }
        this.filteredRepos = this.repositories.filter(repo => repo.fullname.toLowerCase().indexOf(query.toLowerCase()) !== -1);
        this._cd.markForCheck();
    }

    trackRepo(idx: number, r: Repository): string { return r.name; }

    addRepositoryOnProject(): void {
        if (!this.selectedRepo) {
            return;
        }
        let repo = new ProjectRepository();
        repo.name = this.selectedRepo.fullname;
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.addVCSRepository(this.project.key, this.vcsProject.name, repo).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(r => {
            this._toastService.success('Repository added', '');
            this._router.navigate(['/', 'projectv2', this.project.key, 'explore', 'vcs', this.vcsProject.name, 'repository', r.name]).then()
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT
}