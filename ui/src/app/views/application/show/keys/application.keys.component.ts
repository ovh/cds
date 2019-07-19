import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddApplicationKey, DeleteApplicationKey } from 'app/store/applications.action';
import { finalize } from 'rxjs/operators';
import { Application } from '../../../../model/application.model';
import { Project } from '../../../../model/project.model';
import { KeyEvent } from '../../../../shared/keys/key.event';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-keys',
    templateUrl: './application.keys.html',
    styleUrls: ['./application.keys.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ApplicationKeysComponent {

    @Input() project: Project;
    @Input() application: Application;

    loading = false;

    constructor(
        private _toast: ToastService,
        private _translate: TranslateService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {

    }

    manageKeyEvent(event: KeyEvent): void {
        switch (event.type) {
            case 'add':
                this.loading = true;
                this.store.dispatch(new AddApplicationKey({
                    projectKey: this.project.key,
                    applicationName: this.application.name,
                    key: event.key
                })).pipe(finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_added')));
                break;
            case 'delete':
                this.loading = true;
                this.store.dispatch(new DeleteApplicationKey({
                    projectKey: this.project.key,
                    applicationName: this.application.name,
                    key: event.key
                })).pipe(finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_removed')));
        }
    }
}
