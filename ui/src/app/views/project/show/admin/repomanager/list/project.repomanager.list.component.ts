import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { APIConfig } from 'app/model/config.service';
import { Project } from 'app/model/project.model';
import { RepositoriesManager } from 'app/model/repositories.model';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { ConfigState } from 'app/store/config.state';
import { DisconnectRepositoryManagerInProject } from 'app/store/project.action';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-repomanager-list',
    templateUrl: './project.repomanager.list.html',
    styleUrls: ['./project.repomanager.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectRepoManagerComponent implements OnInit {

    @Input() project: Project;
    @Input() reposmanagers: RepositoriesManager[];

    public deleteLoading = false;
    loadingDependencies = false;
    repoNameToDelete: string;
    confirmationMessage: string;
    deleteModal: boolean;
    apiConfig: APIConfig;
    configSubscription: Subscription;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private repoManagerService: RepoManagerService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        this.configSubscription = this._store.select(ConfigState.api).subscribe(c => {
            this.apiConfig = c;
            this._cd.markForCheck();
        });
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
        this._store.dispatch(new DisconnectRepositoryManagerInProject({
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
