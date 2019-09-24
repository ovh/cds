import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { ModelPattern } from 'app/model/worker-model.model';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-pattern-edit',
    templateUrl: './worker-model-pattern.edit.html',
    styleUrls: ['./worker-model-pattern.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelPatternEditComponent {
    loading: boolean;
    pattern: ModelPattern;
    path: Array<PathItem>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) {
        this.loading = true;
        this._workerModelService.getPattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'])
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(p => {
                this.pattern = p;
                this.updatePath();
            });
    }

    onSave(data: ModelPattern): void {
        this.loading = true;
        this._workerModelService.updatePattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'], data)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(p => {
                this.pattern = p;
                this._toast.success('', this._translate.instant('worker_model_pattern_saved'));
                this._router.navigate(['admin', 'worker-model-pattern', p.type, p.name]);
            });
    }

    onDelete() {
        this.loading = true;
        this._workerModelService.deletePattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'])
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('worker_model_pattern_deleted'));
                this._router.navigate(['admin', 'worker-model-pattern']);
            });
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
