import { Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { Project } from 'app/model/project.model';
import { KeyEvent } from 'app/shared/keys/key.event';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddEnvironmentKey, DeleteEnvironmentKey } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

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
