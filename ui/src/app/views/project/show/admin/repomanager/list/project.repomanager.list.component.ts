import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { RepoManagerService } from 'app/service/services.module';
import { ConfirmModalComponent } from 'app/shared/modal/confirm/confirm.component';
import { DisconnectRepositoryManagerInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import { Project } from '../../../../../../model/project.model';
import { RepositoriesManager } from '../../../../../../model/repositories.model';
import { Warning } from '../../../../../../model/warning.model';
import { WarningModalComponent } from '../../../../../../shared/modal/warning/warning.component';
import { Table } from '../../../../../../shared/table/table';
import { ToastService } from '../../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-repomanager-list',
    templateUrl: './project.repomanager.list.html',
    styleUrls: ['./project.repomanager.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectRepoManagerComponent extends Table<RepositoriesManager> {

    @Input() warnings: Map<string, Warning>;
    @Input() project: Project;
    @Input() reposmanagers: RepositoriesManager[];

    @ViewChild('deleteRepoWarning', { static: false })
    private deleteRepoWarning: WarningModalComponent;
    @ViewChild('confirmDeletionModal', { static: false })
    confirmDeletionModal: ConfirmModalComponent;

    public deleteLoading = false;
    loadingDependencies = false;
    repoNameToDelete: string;
    confirmationMessage: string;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private repoManagerService: RepoManagerService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {
        super();
    }

    getData(): Array<RepositoriesManager> {
        return this.reposmanagers;
    }

    clickDeleteButton(repoName: string, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.deleteRepoWarning.show(repoName);
        } else {
            this.loadingDependencies = true;
            this.repoNameToDelete = repoName;
            this.confirmDeletionModal.show();
            this.repoManagerService.getDependencies(this.project.key, repoName)
                .pipe(finalize(() => {
                    this.loadingDependencies = false;
                    this._cd.markForCheck();
                }))
                .subscribe((apps) => {
                    if (!apps) {
                        this.confirmationMessage = this._translate.instant('repoman_delete_confirm_message');
                        return;
                    }
                    this.confirmationMessage = this._translate.instant('repoman_delete_dependencies_message', {
                        apps: apps.map((app) => app.name).join(', ')
                    });
                });
        }

    }

    confirmDeletion(confirm: boolean) {
        if (!confirm) {
            return;
        }
        this.deleteLoading = true;
        this.store.dispatch(new DisconnectRepositoryManagerInProject({ projectKey: this.project.key, repoManager: this.repoNameToDelete }))
            .pipe(finalize(() => {
                this.deleteLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('repoman_delete_msg_ok')));
    }
}
