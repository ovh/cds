import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { VCSProject } from 'app/model/vcs.model';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';
import {
    CallbackRepositoryManagerBasicAuthInProject,
    CallbackRepositoryManagerInProject
} from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-repomanager-form',
    templateUrl: './repomanager.form.html',
    styleUrls: ['./repomanager.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RepoManagerFormComponent {
    @Input() project: Project;
    @Input() disableLabel: boolean = false;

    loading: boolean;
    public ready = false;
    public connectLoading = false;
    public verificationLoading = false;

    // Repo manager form data
    reposManagerList: string[];
    selectedRepoId: number;
    selectedRepoType: string;

    // Repo manager validation
    public addRepoResponse: any;
    validationToken: string;

    basicUser: string;
    basicPassword: string;

    repoModalVisible: boolean;
    addingVCSProject: boolean;

    vcsProject: VCSProject;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _toastService: ToastService,
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private store: Store) {
            this.reposManagerList = ["bitbucketcloud", "bitbucketserver", "github", "gitlab", "gitea", "gerrit"];
    }

    create(): void {
        this.vcsProject = new VCSProject();
        if (this.reposManagerList[this.selectedRepoId]) {
            this.repoModalVisible = true;
            this.selectedRepoType = this.reposManagerList[this.selectedRepoId];
        }
    }

    saveVCSProject(): void {
        if (!this.reposManagerList[this.selectedRepoId]) {
            return;
        }
        
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.addVCSProject(this.project.key, this.vcsProject).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(r => {
            this._toastService.success('Repository Manager updated', '');
            this.repoModalVisible = false;
        });
    }

    sendBasicAuth(): void {
        this.verificationLoading = true;
        this.store.dispatch(new CallbackRepositoryManagerBasicAuthInProject({
            projectKey: this.project.key,
            repoManager: this.reposManagerList[this.selectedRepoId],
            basicUser: this.basicUser,
            basicPassword: this.basicPassword
        }))
            .pipe(finalize(() => {
                this.verificationLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this.repoModalVisible = false;
                this.basicUser = '';
                this.basicPassword = '';
                this._toast.success('', this._translate.instant('repoman_verif_msg_ok'));
            });
    }

    sendVerificationCode(): void {
        this.verificationLoading = true;
        this.store.dispatch(new CallbackRepositoryManagerInProject({
            projectKey: this.project.key,
            repoManager: this.reposManagerList[this.selectedRepoId],
            requestToken: this.addRepoResponse.request_token,
            code: this.validationToken
        })).pipe(finalize(() => {
            this.verificationLoading = false;
            this.repoModalVisible = false;
            this._cd.markForCheck();
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('repoman_verif_msg_ok'));
        });
    }
}
