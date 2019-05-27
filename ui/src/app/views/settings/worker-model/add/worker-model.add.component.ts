import { Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { forkJoin } from 'rxjs/internal/observable/forkJoin';
import { finalize } from 'rxjs/operators';
import { Group } from '../../../../model/group.model';
import { User } from '../../../../model/user.model';
import { ModelPattern, WorkerModel } from '../../../../model/worker-model.model';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
import { GroupService } from '../../../../service/group/group.service';
import { WorkerModelService } from '../../../../service/worker-model/worker-model.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-worker-model-add',
    templateUrl: './worker-model.add.html',
    styleUrls: ['./worker-model.add.scss']
})
export class WorkerModelAddComponent implements OnInit {
    loading = false;
    workerModel: WorkerModel;
    types: Array<string>;
    communications: Array<string>;
    groups: Array<Group>;
    patterns: Array<ModelPattern>;
    patternSelected: ModelPattern;
    currentUser: User;
    path: Array<PathItem>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _groupService: GroupService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _authentificationStore: AuthentificationStore
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
        this.currentUser = this._authentificationStore.getUser();
        this.getGroups();
        this.getWorkerModelComponents();
    }

    getGroups() {
        this.loading = true;
        this._groupService.getGroups()
            .pipe(finalize(() => this.loading = false))
            .subscribe(gs => {
                this.groups = gs;
            });
    }

    getWorkerModelComponents() {
        this.loading = true;
        forkJoin([
            this._workerModelService.getPatterns(),
            this._workerModelService.getTypes(),
            this._workerModelService.getCommunications()
        ])
            .pipe(finalize(() => this.loading = false))
            .subscribe(results => {
                this.patterns = results[0];
                this.types = results[1];
                this.communications = results[2];
            });
    }

    saveWorkerModel(workerModel: WorkerModel): void {
        this.loading = true;
        this._workerModelService.add(workerModel)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('worker_model_saved'));
                this._router.navigate(['settings', 'worker-model', wm.group.name, wm.name]);
            });
    }

    saveWorkerModelAsCode(workerModel: string): void {
        this.loading = true;
        this._workerModelService.import(workerModel, false)
            .pipe(finalize(() => this.loading = false))
            .subscribe((wm) => {
                this.workerModel = wm;
                this._router.navigate(['settings', 'worker-model', this.workerModel.group.name, this.workerModel.name]);
            });
    }
}
