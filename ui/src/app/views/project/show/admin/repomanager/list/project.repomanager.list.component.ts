import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
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
    styleUrls: ['./project.repomanager.list.scss']
})
export class ProjectRepoManagerComponent extends Table<RepositoriesManager> {

    @Input() warnings: Map<string, Warning>;
    @Input() project: Project;
    @Input() reposmanagers: RepositoriesManager[];

    @ViewChild('deleteRepoWarning', {static: false})
    private deleteRepoWarning: WarningModalComponent;

    public deleteLoading = false;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private store: Store
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
            this.deleteLoading = true;
            this.store.dispatch(new DisconnectRepositoryManagerInProject({ projectKey: this.project.key, repoManager: repoName }))
                .pipe(finalize(() => this.deleteLoading = false))
                .subscribe(() => this._toast.success('', this._translate.instant('repoman_delete_msg_ok')));
        }

    }
}
