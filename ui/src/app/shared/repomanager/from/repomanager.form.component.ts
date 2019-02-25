import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { CallbackRepositoryManagerInProject, ConnectRepositoryManagerInProject } from 'app/store/project.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { finalize, flatMap } from 'rxjs/operators';
import { Project } from '../../../model/project.model';
import { RepoManagerService } from '../../../service/repomanager/project.repomanager.service';
import { WarningModalComponent } from '../../modal/warning/warning.component';
import { ToastService } from '../../toast/ToastService';

@Component({
    selector: 'app-repomanager-form',
    templateUrl: './repomanager.form.html',
    styleUrls: ['./repomanager.form.scss']
})
export class RepoManagerFormComponent {

    // project
    @Input() project: Project;

    // Warning modal
    @ViewChild('linkRepoWarning')
    linkRepoWarningModal: WarningModalComponent;

    public ready = false;
    public connectLoading = false;
    public verificationLoading = false;

    // Repo manager form data
    reposManagerList: string[];
    selectedRepoId: number;

    // Repo manager validation
    public addRepoResponse: any;
    validationToken: string;
    private modalInstance: any;

    constructor(
        private _repoManService: RepoManagerService,
        private _toast: ToastService,
        public _translate: TranslateService,
        private store: Store
    ) {
        this._repoManService.getAll().subscribe(res => {
            this.ready = true;
            this.reposManagerList = res;
        });
    }

    create(verificationModal: any, skip?: boolean): void {
        if (this.selectedRepoId && this.reposManagerList[this.selectedRepoId]) {
            if (!skip && this.project.externalChange) {
                this.linkRepoWarningModal.show();
            } else {
                this.connectLoading = true;
                this.store.dispatch(new ConnectRepositoryManagerInProject({
                    projectKey: this.project.key,
                    repoManager: this.reposManagerList[this.selectedRepoId]
                })).pipe(
                    flatMap(() => this.store.selectOnce(ProjectState)),
                    finalize(() => this.connectLoading = false)
                ).subscribe((projState: ProjectStateModel) => {
                    this.addRepoResponse = projState.repoManager;
                    this.modalInstance = verificationModal;
                    setTimeout(() => {
                        verificationModal.show();
                    }, 1);
                });
            }
        }
    }

    sendVerificationCode(): void {
        this.verificationLoading = true;
        this.store.dispatch(new CallbackRepositoryManagerInProject({
            projectKey: this.project.key,
            repoManager: this.reposManagerList[this.selectedRepoId],
            requestToken: this.addRepoResponse.request_token,
            code: this.validationToken
        })).pipe(finalize(() => this.verificationLoading = false))
            .subscribe(() => {
                this.modalInstance.hide();
                this._toast.success('', this._translate.instant('repoman_verif_msg_ok'));
            });
    }
}
