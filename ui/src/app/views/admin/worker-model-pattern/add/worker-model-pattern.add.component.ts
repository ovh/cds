import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import omit from 'lodash-es/omit';
import { finalize } from 'rxjs/operators';
import { User } from '../../../../model/user.model';
import { ModelPattern } from '../../../../model/worker-model.model';
import { WorkerModelService } from '../../../../service/worker-model/worker-model.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-worker-model-pattern-add',
    templateUrl: './worker-model-pattern.add.html',
    styleUrls: ['./worker-model-pattern.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelPatternAddComponent {
    loading = false;
    addLoading = false;
    pattern: ModelPattern;
    workerModelTypes: Array<string>;
    currentUser: User;
    envNames: Array<string> = [];
    newEnvName: string;
    newEnvValue: string;
    path: Array<PathItem>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _toast: ToastService,
        private _translate: TranslateService,
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

        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'worker_model_pattern_title',
            routerLink: ['/', 'admin', 'worker-model-pattern']
        }, <PathItem>{
            translate: 'common_create'
        }];
    }

    clickSaveButton(): void {
        if (this.addLoading || !this.pattern || !this.pattern.name) {
            return;
        }

        this.addLoading = true;
        this._workerModelService.createWorkerModelPattern(this.pattern)
            .pipe(finalize(() => {
                this.addLoading = false;
                this._cd.markForCheck();
            }))
            .subscribe((pattern) => {
                this._toast.success('', this._translate.instant('worker_model_pattern_saved'));
                this._router.navigate(['admin', 'worker-model-pattern', pattern.type, pattern.name]);
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
}
