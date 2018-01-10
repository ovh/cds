import {Component, Input} from '@angular/core';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {Application} from '../../../../model/application.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {Project} from '../../../../model/project.model';
import {finalize} from 'rxjs/operators';
import {KeyEvent} from '../../../../shared/keys/key.event';

@Component({
    selector: 'app-application-keys',
    templateUrl: './application.keys.html',
    styleUrls: ['./application.keys.scss']
})
export class ApplicationKeysComponent {

    @Input() project: Project;
    @Input() application: Application;

    loading = false;

    constructor(private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService) {
    }

    manageKeyEvent(event: KeyEvent): void {
        switch (event.type) {
            case 'add':
                this.loading = true;
                this._appStore.addKey(this.project.key, this.application.name, event.key).pipe(finalize(() => {
                    this.loading = false;
                })).subscribe(() => this._toast.success('', this._translate.instant('keys_added')));
                break;
            case 'delete':
                this.loading = true;
                this._appStore.removeKey(this.project.key, this.application.name, event.key).pipe(finalize(() => {
                    this.loading = false;
                })).subscribe(() => this._toast.success('', this._translate.instant('keys_removed')))
        }
    }
}
