import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ConnectVcsRepoOnApplication, DeleteVcsRepoOnApplication, UpdateApplication } from 'app/store/applications.action';
import { finalize, first } from 'rxjs/operators';
import { Application } from '../../../../../model/application.model';
import { Project } from '../../../../../model/project.model';
import { Repository } from '../../../../../model/repositories.model';
import { RepoManagerService } from '../../../../../service/repomanager/project.repomanager.service';
import { WarningModalComponent } from '../../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-repo',
    templateUrl: './application.repo.html',
    styleUrls: ['./application.repo.scss']
})
export class ApplicationRepositoryComponent implements OnInit {

    @Input() project: Project;
    @Input() application: Application;

    selectedRepoManager: string;
    selectedRepo: string;
    public loadingRepos = false;
    public loadingBtn = false;

    repos: Repository[];
    reposFiltered: Repository[];
    model: string;
    displayVCSStrategy = false;

    @ViewChild('removeWarning', {static: false}) removeWarningModal: WarningModalComponent;
    @ViewChild('linkWarning', {static: false}) linkWarningModal: WarningModalComponent;

    constructor(
        private _repoManagerService: RepoManagerService,
        private _toast: ToastService,
        public _translate: TranslateService,
        private store: Store
    ) {

    }

    ngOnInit() {
        if (this.project.vcs_servers && this.project.vcs_servers.length > 0) {
            this.selectedRepoManager = this.project.vcs_servers[0].name;
        }
        this.displayVCSStrategy = !this.application.vcs_strategy || !this.application.vcs_strategy.connection_type;
        this.updateListRepo(false);
    }

    removeRepository(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.removeWarningModal.show();
        } else {
            this.loadingBtn = true;
            this.store.dispatch(new DeleteVcsRepoOnApplication({
                projectKey: this.project.key,
                applicationName: this.application.name,
                repoManager: this.application.vcs_server
            })).pipe(finalize(() => this.loadingBtn = false))
                .subscribe(() => this._toast.success('', this._translate.instant('application_repo_detach_ok')));
        }
    }

    filterRepositories(filter: string): void {
        if (filter.length >= 3) {
            this.reposFiltered = this.repos.filter(r => {
                return r.fullname.toLowerCase().indexOf(filter.toLowerCase()) !== -1;
            });
        } else {
            this.reposFiltered = this.repos.slice(0, 50);
        }
    }

    /**
     * Update list of repo when changing repo manager
     */
    updateListRepo(sync: boolean): void {
        if (this.selectedRepoManager) {
            this.loadingRepos = true;
            this._repoManagerService.getRepositories(this.project.key, this.selectedRepoManager, sync).pipe(first())
                .subscribe(repos => {
                    this.repos = repos;
                    this.reposFiltered = repos.slice(0, 50);
                },
                    null,
                    () => this.loadingRepos = false
                );
        }
    }

    linkRepository(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.linkWarningModal.show();
        } else {
            this.loadingBtn = true;
            this.store.dispatch(new ConnectVcsRepoOnApplication({
                projectKey: this.project.key,
                applicationName: this.application.name,
                repoManager: this.selectedRepoManager,
                repoFullName: this.selectedRepo
            })).pipe(finalize(() => this.loadingBtn = false))
                .subscribe(() => {
                    this.displayVCSStrategy = !this.application.vcs_strategy || !this.application.vcs_strategy.connection_type;
                    this._toast.success('', this._translate.instant('application_repo_attach_ok'));
                });
        }
    }

    saveVCSConfiguration(): void {
        this.loadingBtn = true;
        this.store.dispatch(new UpdateApplication({
            projectKey: this.project.key,
            applicationName: this.application.name,
            changes: this.application
        })).pipe(finalize(() => this.loadingBtn = false))
            .subscribe(() => this._toast.success('', this._translate.instant('application_update_ok')));
    }
}
