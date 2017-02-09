import {Component, Input, ViewChild} from '@angular/core';
import {ApplicationStore} from '../../../../../../service/application/application.store';
import {TranslateService} from 'ng2-translate/ng2-translate';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {Table} from '../../../../../../shared/table/table';
import {Project} from '../../../../../../model/project.model';
import {Application} from '../../../../../../model/application.model';
import {Hook} from '../../../../../../model/hook.model';
import {WarningModalComponent} from '../../../../../../shared/modal/warning/warning.component';

@Component({
    selector: 'app-application-hook-list',
    templateUrl: './application.hook.list.html',
    styleUrls: ['./application.hook.list.scss']
})
export class ApplicationHookListComponent extends Table {

    @Input() project: Project;
    @Input() application: Application;
    @ViewChild('deleteHookWarning') deleteWarningModal: WarningModalComponent;

    public loading = false;

    getData(): any[] {
        return this.application.hooks;
    }

    constructor(private _toast: ToastService, public _translate: TranslateService, private _appStore: ApplicationStore) {
        super();

    }

    deleteHook(h: Hook, skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.deleteWarningModal.show(h);
        } else {
            this.loading = true;
            this._appStore.removeHook(this.project, this.application, h).subscribe(() => {
                this._toast.success('', this._translate.instant('application_hook_delete_ok'));
                this.loading = false;
            }, () => {
                this.loading = false;
            });
        }
    }
}
