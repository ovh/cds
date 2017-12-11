import {Component, Input, ViewChild} from '@angular/core';
import {RepositoriesManager} from '../../../../../../model/repositories.model';
import {Table} from '../../../../../../shared/table/table';
import {TranslateService} from '@ngx-translate/core';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {Project} from '../../../../../../model/project.model';
import {WarningModalComponent} from '../../../../../../shared/modal/warning/warning.component';

@Component({
    selector: 'app-project-repomanager-list',
    templateUrl: './project.repomanager.list.html',
    styleUrls: ['./project.repomanager.list.scss']
})
export class ProjectRepoManagerComponent extends Table {

    @Input() project: Project;
    @Input() reposmanagers: RepositoriesManager[];

    @ViewChild('deleteRepoWarning')
    private deleteRepoWarning: WarningModalComponent;

    public deleteLoading = false;

    constructor(private _toast: ToastService, public _translate: TranslateService, private _projectStore: ProjectStore) {
        super();
    }

    getData() {
        return this.reposmanagers;
    }

    clickDeleteButton(repoName: string, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.deleteRepoWarning.show(repoName);
        } else {
            this.deleteLoading = true;
            this._projectStore.disconnectRepoManager(this.project.key, repoName).subscribe(() => {
                this._toast.success('', this._translate.instant('repoman_delete_msg_ok'));
                this.deleteLoading = false;
            }, () => {
                this.deleteLoading = false;
            });
        }

    }
}
