import {Component, Input, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Project} from '../../../../../../model/project.model';
import {RepositoriesManager} from '../../../../../../model/repositories.model';
import {Warning} from '../../../../../../model/warning.model';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {WarningModalComponent} from '../../../../../../shared/modal/warning/warning.component';
import {Table} from '../../../../../../shared/table/table';
import {ToastService} from '../../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-repomanager-list',
    templateUrl: './project.repomanager.list.html',
    styleUrls: ['./project.repomanager.list.scss']
})
export class ProjectRepoManagerComponent extends Table {

    @Input() warnings: Map<string, Warning>;
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
