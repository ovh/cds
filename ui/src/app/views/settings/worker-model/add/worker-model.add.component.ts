import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import { forkJoin } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Group } from '../../../../model/group.model';
import { AuthSummary } from '../../../../model/user.model';
import { ModelPattern, WorkerModel } from '../../../../model/worker-model.model';
import { GroupService } from '../../../../service/group/group.service';
import { WorkerModelService } from '../../../../service/worker-model/worker-model.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-worker-model-add',
    templateUrl: './worker-model.add.html',
    styleUrls: ['./worker-model.add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelAddComponent implements OnInit {
    loading = false;
    workerModel: WorkerModel;
    types: Array<string>;
    groups: Array<Group>;
    patterns: Array<ModelPattern>;
    patternSelected: ModelPattern;
    currentAuthSummary: AuthSummary;
    path: Array<PathItem>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _groupService: GroupService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'worker_model_list_title',
            routerLink: ['/', 'settings', 'worker-model']
        }, <PathItem>{
            translate: 'common_create'
        }];

        this.workerModel = new WorkerModel();
        this.workerModel.editable = true;
        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);
        this.getGroups();
        this.getWorkerModelComponents();
    }

    getGroups() {
        this.loading = true;
        this._groupService.getAll()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    getWorkerModelComponents() {
        this.loading = true;
        forkJoin([
            this._workerModelService.getPatterns(),
            this._workerModelService.getTypes()
        ])
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(results => {
                this.patterns = results[0];
                this.types = results[1];
            });
    }

    saveWorkerModel(workerModel: WorkerModel): void {
        this.loading = true;
        this._workerModelService.add(workerModel)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('worker_model_saved'));
                this._router.navigate(['settings', 'worker-model', wm.group.name, wm.name]);
            });
    }

    saveWorkerModelAsCode(workerModel: string): void {
        this.loading = true;
        this._workerModelService.import(workerModel, false)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((wm) => {
                this.workerModel = wm;
                this._router.navigate(['settings', 'worker-model', this.workerModel.group.name, this.workerModel.name]);
            });
    }
}
