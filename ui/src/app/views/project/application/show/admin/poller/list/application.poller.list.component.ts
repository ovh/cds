import {Component, Input, ViewChild} from '@angular/core';
import {ApplicationStore} from '../../../../../../../service/application/application.store';
import {TranslateService} from 'ng2-translate/ng2-translate';
import {ToastService} from '../../../../../../../shared/toast/ToastService';
import {Table} from '../../../../../../../shared/table/table';
import {RepositoryPoller} from '../../../../../../../model/polling.model';
import {Project} from '../../../../../../../model/project.model';
import {Application} from '../../../../../../../model/application.model';
import {WarningModalComponent} from '../../../../../../../shared/modal/warning/warning.component';

@Component({
    selector: 'app-application-poller-list',
    templateUrl: './application.poller.list.html',
    styleUrls: ['./application.poller.list.scss']
})
export class ApplicationPollerListComponent extends Table {

    @Input() project: Project;
    @Input() application: Application;
    @ViewChild('deletePollerWarning') warningModal: WarningModalComponent;

    public loading = false;

    getData(): any[] {
        return this.application.pollers;
    }

    constructor(private _toast: ToastService, public _translate: TranslateService, private _appStore: ApplicationStore) {
        super();

    }

    deletePollers(p: RepositoryPoller, skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.warningModal.show(p);
        } else {
            this.loading = true;
            this._appStore.deletePoller(this.project.key, this.application.name, p).subscribe(() => {
                this._toast.success('', this._translate.instant('application_poller_delete_ok'));
                this.loading = false;
            }, () => {
                this.loading = false;
            });
        }
    }


}
