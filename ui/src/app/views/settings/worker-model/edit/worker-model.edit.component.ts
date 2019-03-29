import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { forkJoin } from 'rxjs/internal/observable/forkJoin';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Group } from '../../../../model/group.model';
import { Pipeline } from '../../../../model/pipeline.model';
import { User } from '../../../../model/user.model';
import { ModelPattern, WorkerModel } from '../../../../model/worker-model.model';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
import { GroupService } from '../../../../service/group/group.service';
import { WorkerModelService } from '../../../../service/worker-model/worker-model.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { Tab } from '../../../../shared/tabs/tabs.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-worker-model-edit',
    templateUrl: './worker-model.edit.html',
    styleUrls: ['./worker-model.edit.scss']
})
@AutoUnsubscribe()
export class WorkerModelEditComponent implements OnInit {
    loading = false;
    loadingUsage = false;
    workerModel: WorkerModel;
    types: Array<string>;
    communications: Array<string>;
    groups: Array<Group>;
    patterns: Array<ModelPattern>;
    currentUser: User;
    usages: Array<Pipeline>;
    path: Array<PathItem>;
    paramsSub: Subscription;
    tabs: Array<Tab>;
    selectedTab: Tab;
    workerModelName: string;

    constructor(
        private _workerModelService: WorkerModelService,
        private _groupService: GroupService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _authentificationStore: AuthentificationStore
    ) { }

    selectTab(tab: Tab): void {
        switch (tab.key) {
            case 'usage':
                this.loadUsage();
                break;
        }
        this.selectedTab = tab;
    }

    ngOnInit() {
        this.tabs = [<Tab>{
            translate: 'worker_model',
            icon: '',
            key: 'worker_model',
            default: true
        }, <Tab>{
            translate: 'common_capabilities',
            icon: 'file outline',
            key: 'capabilities'
        }, <Tab>{
            translate: 'common_usage',
            icon: 'map signs',
            key: 'usage'
        }];

        this.currentUser = this._authentificationStore.getUser();
        this.getGroups();
        this.getWorkerModelComponents();

        this.paramsSub = this._route.params.subscribe(params => {
            this.workerModelName = params['workerModelName'];
            this.getWorkerModel(this.workerModelName);
        });
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
            this._workerModelService.getWorkerModelPatterns(),
            this._workerModelService.getWorkerModelTypes(),
            this._workerModelService.getWorkerModelCommunications()
        ])
            .pipe(finalize(() => this.loading = false))
            .subscribe(results => {
                this.patterns = results[0];
                this.types = results[1];
                this.communications = results[2];
            });
    }

    getWorkerModel(workerModelName: string): void {
        this.loading = true;
        this._workerModelService.getWorkerModelByName(workerModelName)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wm => {
                this.getWorkerModelPermission(wm);
                this.updatePath();
            });
    }

    // FIXME calculate editable value, should be done by the api
    getWorkerModelPermission(workerModel: WorkerModel): void {
        if (this.currentUser.admin) {
            workerModel.editable = true;
            this.workerModel = workerModel;
            this.updatePath();
            return;
        }

        if (!this.currentUser.admin && workerModel.group.name === 'shared.infra') {
            workerModel.editable = false;
            this.workerModel = workerModel;
            this.updatePath();
            return;
        }

        // here, check if user is admin of worker model group
        this._groupService.getGroupByName(workerModel.group.name).subscribe(g => {
            if (g.admins) {
                for (let i = 0; i < g.admins.length; i++) {
                    if (g.admins[i].username === this.currentUser.username) {
                        workerModel.editable = true;
                        break;
                    };
                }
            }
            this.workerModel = workerModel;
            this.updatePath();
        });
    }

    deleteWorkerModel(): void {
        this.loading = true;
        this._workerModelService.deleteWorkerModel(this.workerModel)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('worker_model_deleted'));
                this._router.navigate(['../'], { relativeTo: this._route });
            });
    }

    saveWorkerModel(workerModel: WorkerModel) {
        this.loading = true;
        this._workerModelService.updateWorkerModel(workerModel)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wm => {
                this.getWorkerModelPermission(wm);
                this._toast.success('', this._translate.instant('worker_model_saved'));
                this._router.navigate(['settings', 'worker-model', wm.name]);
            });
    }

    saveWorkerModelAsCode(workerModel: string): void {
        this.loading = true;
        this._workerModelService.importWorkerModel(workerModel, true)
            .pipe(finalize(() => this.loading = false))
            .subscribe((wm) => {
                this.getWorkerModelPermission(wm);
                this._toast.success('', this._translate.instant('worker_model_saved'));
                this._router.navigate(['settings', 'worker-model', wm.name]);
            });
    }

    loadUsage() {
        // FIXME model endpoint should take path not id
        if (!this.workerModel || !this.workerModel.id) {
            this._router.navigate([], {
                relativeTo: this._route,
                queryParams: { tab: this.tabs[0].key },
                queryParamsHandling: 'merge'
            });
            return;
        }

        this.loadingUsage = true;
        this._workerModelService.getUsage(this.workerModel.id)
            .pipe(finalize(() => this.loadingUsage = false))
            .subscribe((usages) => {
                this.usages = usages;
            });
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'worker_model_list_title',
            routerLink: ['/', 'settings', 'worker-model']
        }];

        if (this.workerModel && this.workerModel.id) {
            this.path.push(<PathItem>{
                text: this.workerModel.name,
                routerLink: ['/', 'settings', 'worker-model', this.workerModel.name]
            });
        }
    }
}
