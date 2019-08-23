import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthentifiedUser } from 'app/model/user.model';
import { ModelPattern } from 'app/model/worker-model.model';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import omit from 'lodash-es/omit';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-pattern-edit',
    templateUrl: './worker-model-pattern.edit.html',
    styleUrls: ['./worker-model-pattern.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelPatternEditComponent implements OnInit {
    loading = false;
    editLoading = false;
    pattern: ModelPattern;
    workerModelTypes: Array<string>;
    currentUser: AuthentifiedUser;
    envNames: Array<string> = [];
    newEnvName: string;
    newEnvValue: string;
    path: Array<PathItem>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
        this.loading = true;
        this._workerModelService.getTypes()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wmt => this.workerModelTypes = wmt);
        this.pattern = new ModelPattern();
    }

    ngOnInit() {
        this.loading = true;
        this._workerModelService.getPattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'])
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(
                (pattern) => {
                    if (pattern.model.envs) {
                        this.envNames = Object.keys(pattern.model.envs);
                    }
                    this.pattern = pattern;
                    this.updatePath();
                },
                () => this._router.navigate(['admin', 'worker-model-pattern'])
            );
    }

    clickSaveButton(): void {
        if (!this.pattern || !this.pattern.name) {
            return;
        }

        this.editLoading = true;
        this._workerModelService
            .updatePattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'], this.pattern)
            .pipe(finalize(() => {
                this.editLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe((pattern) => {
                this._toast.success('', this._translate.instant('worker_model_pattern_saved'));
                this._router.navigate(['admin', 'worker-model-pattern', pattern.type, pattern.name]);
            });
    }

    delete() {
        if (this.editLoading) {
            return;
        }

        this.editLoading = true;
        this._workerModelService
            .deletePattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'])
            .pipe(finalize(() => {
                this.editLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('worker_model_pattern_deleted'));
                this._router.navigate(['admin', 'worker-model-pattern']);
            });
    }

    addEnv(newEnvName: string, newEnvValue: string) {
        if (!newEnvName) {
            return;
        }
        if (!this.pattern.model.envs) {
            this.pattern.model.envs = {};
        }
        this.pattern.model.envs[newEnvName] = newEnvValue;
        this.envNames.push(newEnvName);
        this.newEnvName = '';
        this.newEnvValue = '';
    }

    deleteEnv(envName: string, index: number) {
        this.envNames.splice(index, 1);
        this.pattern.model.envs = omit(this.pattern.model.envs, envName);
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'worker_model_pattern_title',
            routerLink: ['/', 'admin', 'worker-model-pattern']
        }];

        if (this.pattern && this.pattern.name) {
            this.path.push(<PathItem>{
                text: this.pattern.name,
                routerLink: ['/', 'admin', 'worker-model-pattern', this.pattern.name]
            });
        }
    }
}
