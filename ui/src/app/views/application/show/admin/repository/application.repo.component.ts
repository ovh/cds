import {Component, OnInit, Input, ViewChild} from '@angular/core';
import {Application} from '../../../../../model/application.model';
import {RepoManagerService} from '../../../../../service/repomanager/project.repomanager.service';
import {Repository} from '../../../../../model/repositories.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {Project} from '../../../../../model/project.model';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {WarningModalComponent} from '../../../../../shared/modal/warning/warning.component';

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
    reposTmp: Repository[];
    model: string;

    @ViewChild('removeWarning') removeWarningModal: WarningModalComponent;
    @ViewChild('linkWarning') linkWarningModal: WarningModalComponent;

    constructor(private _appStore: ApplicationStore, private _repoManagerService: RepoManagerService,
                private _toast: ToastService, public _translate: TranslateService) {
    }

    ngOnInit() {
        if (this.project.repositories_manager && this.project.repositories_manager.length > 0) {
            this.selectedRepoManager = this.project.repositories_manager[0].name;
        }
        this.updateListRepo();
    }

    removeRepository(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.removeWarningModal.show();
        } else {
            this.loadingBtn = true;
            this._appStore.removeRepository(this.project.key, this.application.name, this.application.repositories_manager.name)
                .subscribe( () => {
                    delete this.application.repositories_manager;
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
            this.reposTmp = this.repos.filter(r => {
                return r.fullname.toLowerCase().indexOf(filter.toLowerCase()) !== -1;
            });
        }
    }

    /**
     * Update list of repo when changing repo manager
     */
    updateListRepo(): void {
        if (this.selectedRepoManager) {
            this.loadingRepos = true;
            this._repoManagerService.getRepositories(this.project.key, this.selectedRepoManager)
                .subscribe( repos => {
                    this.repos = repos;
                    this.loadingRepos = false;
                }, () => {
                    this.loadingRepos = false;
                });
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
