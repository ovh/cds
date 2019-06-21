import { Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddEnvironmentKey, DeleteEnvironmentKey } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import { Environment } from '../../../../model/environment.model';
import { Project } from '../../../../model/project.model';
import { KeyEvent } from '../../../../shared/keys/key.event';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-environment-keys',
    templateUrl: './environment.keys.html',
    styleUrls: ['./environment.keys.scss']
})
export class EnvironmentKeysComponent {

    @Input() project: Project;
    @Input() environment: Environment;

    loading = false;

    constructor(
        private _toast: ToastService,
        private _translate: TranslateService,
        private store: Store
    ) {

    }

    manageKeyEvent(event: KeyEvent): void {
        switch (event.type) {
            case 'add':
                this.loading = true;
                this.store.dispatch(new AddEnvironmentKey({
                    projectKey: this.project.key,
                    envName: this.environment.name,
                    key: event.key
                })).pipe(finalize(() => this.loading = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_added')));
                break;
            case 'delete':
                this.loading = true;
                this.store.dispatch(new DeleteEnvironmentKey({
                    projectKey: this.project.key,
                    envName: this.environment.name,
                    key: event.key
                })).pipe(finalize(() => this.loading = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_removed')));
        }
    }
}
