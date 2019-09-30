import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { ModelPattern } from 'app/model/worker-model.model';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-pattern-add',
    templateUrl: './worker-model-pattern.add.html',
    styleUrls: ['./worker-model-pattern.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelPatternAddComponent {
    loading: boolean;
    pattern: ModelPattern;
    path: Array<PathItem>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) {
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

    onSave(m: ModelPattern): void {
        this.loading = true;
        this._workerModelService.createPattern(m)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((pattern) => {
                this.pattern = m;
                this._toast.success('', this._translate.instant('worker_model_pattern_saved'));
                this._router.navigate(['admin', 'worker-model-pattern', pattern.type, pattern.name]);
            });
    }
}
