import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import { forkJoin } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Group } from '../../../../model/group.model';
import { Pipeline } from '../../../../model/pipeline.model';
import { AuthSummary } from '../../../../model/user.model';
import { ModelPattern, WorkerModel } from '../../../../model/worker-model.model';
import { GroupService } from '../../../../service/group/group.service';
import { WorkerModelService } from '../../../../service/worker-model/worker-model.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { Tab } from '../../../../shared/tabs/tabs.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-worker-model-edit',
    templateUrl: './worker-model.edit.html',
    styleUrls: ['./worker-model.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkerModelEditComponent implements OnInit, OnDestroy {
    loading = false;
    loadingUsage = false;
    workerModel: WorkerModel;
    types: Array<string>;
    groups: Array<Group>;
    patterns: Array<ModelPattern>;
    currentAuthSummary: AuthSummary;
    usages: Array<Pipeline>;
    path: Array<PathItem>;
    paramsSub: Subscription;
    tabs: Array<Tab>;
    selectedTab: Tab;
    groupName: string;
    workerModelName: string;

    constructor(
        private _workerModelService: WorkerModelService,
        private _groupService: GroupService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

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

        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);
        this.getGroups();
        this.getWorkerModelComponents();

        this.paramsSub = this._route.params.subscribe(params => {
            this.groupName = params['groupName'];
            this.workerModelName = params['workerModelName'];
            this.getWorkerModel(this.groupName, this.workerModelName);
            this._cd.markForCheck();
        });
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

    getWorkerModel(groupName: string, modelName: string): void {
        this.loading = true;
        this._workerModelService.get(groupName, modelName)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wm => {
                this.workerModel = wm;
                this.updatePath();
            });
    }

    deleteWorkerModel(): void {
        this.loading = true;
        this._workerModelService.delete(this.groupName, this.workerModelName)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wm => {
                this._toast.success('', this._translate.instant('worker_model_deleted'));
                this._router.navigate(['settings', 'worker-model']);
            });
    }

    saveWorkerModel(workerModel: WorkerModel) {
        this.loading = true;
        this._workerModelService.update(this.workerModel, workerModel)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wm => {
                this.workerModel = wm;
                this.updatePath();
                this._toast.success('', this._translate.instant('worker_model_saved'));
                this._router.navigate(['settings', 'worker-model', wm.group.name, wm.name]);
            });
    }

    saveWorkerModelAsCode(workerModel: string): void {
        this.loading = true;
        this._workerModelService.import(workerModel, true)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((wm) => {
                this.workerModel = wm;
                this.updatePath();
                this._toast.success('', this._translate.instant('worker_model_saved'));
                this._router.navigate(['settings', 'worker-model', wm.group.name, wm.name]);
            });
    }

    loadUsage() {
        this.loadingUsage = true;
        this._workerModelService.getUsage(this.groupName, this.workerModelName)
            .pipe(finalize(() => {
                this.loadingUsage = false;
                this._cd.markForCheck();
            }))
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
                routerLink: ['/', 'settings', 'worker-model', this.workerModel.group.name, this.workerModel.name]
            });
        }
    }
}
