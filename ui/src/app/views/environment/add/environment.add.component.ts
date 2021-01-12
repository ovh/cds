import { ChangeDetectionStrategy, Component, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { Project } from 'app/model/project.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddEnvironment } from 'app/store/environment.action';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-environment-add',
    templateUrl: './environment.add.html',
    styleUrls: ['./environment.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class EnvironmentAddComponent implements OnDestroy {

    project: Project;
    newEnvironment: Environment = new Environment();
    envPatternError = false;
    loading = false;
    dataSubscription: Subscription;
    environmentNamePattern = new RegExp('^[a-zA-Z0-9._-]{1,}$');

    constructor(
        private store: Store,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.dataSubscription = this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    checkPattern(envName: string) {
        if (!envName) {
            return;
        }

        this.envPatternError = !this.environmentNamePattern.test(envName);
    }

    createEnv(): void {
        if (!this.newEnvironment.name || this.envPatternError) {
            return;
        }

        this.loading = true;
        this.store.dispatch(new AddEnvironment({ projectKey: this.project.key, environment: this.newEnvironment }))
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('environment_created'));
                this._router.navigate(['/project', this.project.key, 'environment', this.newEnvironment.name]);
                this.newEnvironment = new Environment();
            });
    }
}
