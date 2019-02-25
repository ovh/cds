import { Component, Input, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddKeyInProject, DeleteKeyInProject, FetchKeysInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import { Key } from '../../../../model/keys.model';
import { Project } from '../../../../model/project.model';
import { Warning } from '../../../../model/warning.model';
import { KeyEvent } from '../../../../shared/keys/key.event';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-keys',
    templateUrl: './project.keys.html',
    styleUrls: ['./project.keys.scss']
})
export class ProjectKeysComponent implements OnInit {

    _project: Project;
    @Input('project')
    set project(data: Project) {
        if (data) {
            this._project = data;
            this.keys = data.keys;
        }
    }

    get project() {
        return this._project;
    }

    keys: Array<Key>;

    @Input('warnings')
    set warnings(data: Array<Warning>) {
        if (data) {
            this.unusedWarning = new Map<string, Warning>();
            this.missingWarnings = new Array<Warning>();
            data.forEach(v => {
                if (v.type.indexOf('MISSING') !== -1) {
                    this.missingWarnings.push(v);
                } else {
                    this.unusedWarning.set(v.element, v);
                }
            });
        }
    };
    missingWarnings: Array<Warning>;
    unusedWarning: Map<string, Warning>;

    loading = false;
    ready = false;

    constructor(
        private _toast: ToastService,
        private _translate: TranslateService,
        private store: Store
    ) {
    }

    ngOnInit(): void {
        this.store.dispatch(new FetchKeysInProject({ projectKey: this.project.key }))
            .pipe(finalize(() => this.ready = true))
            .subscribe();
    }

    manageKeyEvent(event: KeyEvent): void {
        switch (event.type) {
            case 'add':
                this.loading = true;
                this.store.dispatch(new AddKeyInProject({ projectKey: this.project.key, key: event.key }))
                    .pipe(finalize(() => this.loading = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_added')));
                break;
            case 'delete':
                this.loading = true;
                this.store.dispatch(new DeleteKeyInProject({ projectKey: this.project.key, key: event.key }))
                    .pipe(finalize(() => this.loading = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_removed')));
        }
    }
}
