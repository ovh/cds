import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { RepositoriesManager } from 'app/model/repositories.model';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { DisconnectRepositoryManagerInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-repomanager-list',
    templateUrl: './project.repomanager.list.html',
    styleUrls: ['./project.repomanager.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectRepoManagerComponent {

    @Input() project: Project;
    @Input() reposmanagers: RepositoriesManager[];

    public deleteLoading = false;
    loadingDependencies = false;
    repoNameToDelete: string;
    confirmationMessage: string;
    deleteModal: boolean;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private repoManagerService: RepoManagerService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {
    }

    clickDeleteButton(repoName: string): void {
        this.loadingDependencies = true;
        this.repoNameToDelete = repoName;
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
                this.deleteModal = true;
            });
    }

    confirmDeletion(confirm: boolean) {
        if (!confirm) {
            return;
        }
        this.deleteLoading = true;
        this.store.dispatch(new DisconnectRepositoryManagerInProject({
            projectKey: this.project.key,
            repoManager: this.repoNameToDelete
        }))
            .pipe(finalize(() => {
                this.deleteLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('repoman_delete_msg_ok')));
    }
}
