import {Component, ViewChild, Input} from '@angular/core';
import {RepoManagerService} from '../../../../../../service/repomanager/project.repomanager.service';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {Project} from '../../../../../../model/project.model';
import {WarningModalComponent} from '../../../../../../shared/modal/warning/warning.component';

@Component({
    selector: 'app-project-repomanager-form',
    templateUrl: './project.repomanager.form.html',
    styleUrls: ['./project.repomanager.form.scss']
})
export class ProjectRepoManagerFormComponent  {

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

    constructor(private _repoManService: RepoManagerService, private _projectStore: ProjectStore,
                private _toast: ToastService, public _translate: TranslateService) {
        this._repoManService.getAll().subscribe( res => {
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
                this._projectStore.connectRepoManager(this.project.key, this.reposManagerList[this.selectedRepoId])
                  .subscribe( res => {
                      this.connectLoading = false;
                      this.addRepoResponse = res;
                      this.modalInstance = verificationModal;
                      verificationModal.show();
                  }, () => {
                      this.connectLoading = false;
                  });
            }
        }
    }

    sendVerificationCode(): void {
        this.verificationLoading = true;
        this._projectStore.verificationCallBackRepoManager(
            this.project.key, this.reposManagerList[this.selectedRepoId], this.addRepoResponse.request_token, this.validationToken
        ).subscribe( () => {
            this.verificationLoading = false;
            this.modalInstance.hide();
            this._toast.success('', this._translate.instant('repoman_verif_msg_ok'));
        }, () => {
            this.verificationLoading = false;
        });
    }

}
