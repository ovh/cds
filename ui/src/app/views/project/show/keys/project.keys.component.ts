import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Key } from 'app/model/keys.model';
import { Project } from 'app/model/project.model';
import { KeyEvent } from 'app/shared/keys/key.event';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddKeyInProject, DeleteKeyInProject, FetchKeysInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-keys',
    templateUrl: './project.keys.html',
    styleUrls: ['./project.keys.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectKeysComponent implements OnInit {

    _project: Project;
    @Input()
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

    loading = false;
    ready = false;

    constructor(
        private _toast: ToastService,
        private _translate: TranslateService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {
    }

    ngOnInit(): void {
        this.store.dispatch(new FetchKeysInProject({ projectKey: this.project.key }))
            .pipe(finalize(() => {
                this.ready = true;
                this._cd.markForCheck();
            }))
            .subscribe();
    }

    manageKeyEvent(event: KeyEvent): void {
        switch (event.type) {
            case 'add':
                this.loading = true;
                this.store.dispatch(new AddKeyInProject({ projectKey: this.project.key, key: event.key }))
                    .pipe(finalize(() => {
                        this.loading = false;
                        this._cd.markForCheck();
                    }))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_added')));
                break;
            case 'delete':
                this.loading = true;
                this.store.dispatch(new DeleteKeyInProject({ projectKey: this.project.key, key: event.key }))
                    .pipe(finalize(() => {
                        this.loading = false;
                        this._cd.markForCheck();
                    }))
                    .subscribe(() => this._toast.success('', this._translate.instant('keys_removed')));
        }
    }
}
