import {Component, OnInit, Input, ViewChild} from '@angular/core';
import {Application} from '../../../../../model/application.model';
import {RepoManagerService} from '../../../../../service/repomanager/project.repomanager.service';
import {Repository} from '../../../../../model/repositories.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {Project} from '../../../../../model/project.model';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {WarningModalComponent} from '../../../../../shared/modal/warning/warning.component';
import {first} from 'rxjs/operators';

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

    @ViewChild('removeWarning') removeWarningModal: WarningModalComponent;
    @ViewChild('linkWarning') linkWarningModal: WarningModalComponent;

    constructor(private _appStore: ApplicationStore, private _repoManagerService: RepoManagerService,
                private _toast: ToastService, public _translate: TranslateService) {
    }

    ngOnInit() {
        if (this.project.vcs_servers && this.project.vcs_servers.length > 0) {
            this.selectedRepoManager = this.project.vcs_servers[0].name;
        }
        this.updateListRepo(false);
    }

    removeRepository(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.removeWarningModal.show();
        } else {
            this.loadingBtn = true;
            this._appStore.removeRepository(this.project.key, this.application.name, this.application.vcs_server)
                .subscribe( () => {
                    delete this.application.vcs_server;
                    delete this.application.repository_fullname;
                    this.loadingBtn = false;
                    this._toast.success('', this._translate.instant('application_repo_detach_ok'));
                }, () => {
                    this.loadingBtn = false;
                });
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
            this._appStore.connectRepository(this.project.key, this.application.name, this.selectedRepoManager, this.selectedRepo)
                .subscribe(() => {
                    this.loadingBtn = false;
                    this._toast.success('', this._translate.instant('application_repo_attach_ok'));
                }, () => {
                    this.loadingBtn = false;
                });
        }
    }
}
