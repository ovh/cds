import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { LoadOpts, Project } from 'app/model/project.model';
import { HelpersService } from 'app/service/helpers/helpers.service';
import { ProjectStore } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Tab } from 'app/shared/tabs/tabs.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchProject, UpdateFavoriteProject } from 'app/store/project.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectShowComponent implements OnInit, OnDestroy {

    project: Project;
    projectSubscriber: Subscription;

    tabs: Array<Tab>;
    selectedTab: Tab;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;
    loadingFav = false;

    constructor(
        private _route: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _helpersService: HelpersService,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _projectStore: ProjectStore,
    ) {
        this.projectSubscriber = this._store.select(ProjectState)
            .pipe(filter((projState: ProjectStateModel) => projState && projState.project && projState.project.key !== null && !projState.project.externalChange &&
                    this._route.snapshot.params['key'] === projState.project.key))
            .subscribe((projState: ProjectStateModel) => {
                let proj = cloneDeep(projState.project); // TODO: to delete when all will be in store, here it is usefull to skip readonly
                if (proj.labels) {
                    proj.labels = proj.labels.map((lbl) => {
                        lbl.font_color = this._helpersService.getBrightnessColor(lbl.color);
                        return lbl;
                    });
                }
                this.project = proj;
                this._projectStore.updateRecentProject(this.project);
                this._cd.markForCheck();
            });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.tabs = [<Tab>{
            translate: 'common_workflows',
            icon: 'share alternate',
            key: 'workflows',
            default: true
        }, <Tab>{
            translate: 'common_applications',
            icon: 'rocket',
            key: 'applications'
        }, <Tab>{
            translate: 'common_pipelines',
            icon: 'sitemap',
            key: 'pipelines'
        }, <Tab>{
            translate: 'common_environments',
            icon: 'tree',
            key: 'environments'
        }, <Tab>{
            translate: 'common_variables',
            icon: 'font',
            key: 'variables'
        }, <Tab>{
            translate: 'common_permissions',
            icon: 'users',
            key: 'permissions'
        }, <Tab>{
            translate: 'common_keys',
            icon: 'privacy',
            key: 'keys'
        }, <Tab>{
            translate: 'common_integrations',
            icon: 'plug',
            key: 'integrations'
        }, <Tab>{
            translate: 'common_warnings',
            icon: 'bug',
            key: 'warnings'
        }, <Tab>{
            translate: 'common_advanced',
            icon: 'graduation',
            key: 'advanced'
        }];

        this._route.queryParams.subscribe((queryParams) => {
            if (queryParams['tab']) {
                let current_tab = this.tabs.find((tab) => tab.key === queryParams['tab']);
                if (current_tab) {
                    this.selectTab(current_tab);
                }
                this._cd.markForCheck();
            }
            this._route.params.subscribe(routeParams => {
                const key = routeParams['key'];
                if (key) {
                    if (this.project && this.project.key !== key) {
                        this.project = undefined;
                    }
                    if (!this.project) {
                        this.refreshDatas(key);
                    }
                    this._cd.markForCheck();
                }
            });
        });

        if (this._route.snapshot && this._route.snapshot.queryParams) {
            this.workflowName = this._route.snapshot.queryParams['workflow'];
            this.workflowNum = this._route.snapshot.queryParams['run'];
            this.workflowNodeRun = this._route.snapshot.queryParams['node'];
            this.workflowPipeline = this._route.snapshot.queryParams['wpipeline'];
        }
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    refreshDatas(key: string): void {
        let opts = [new LoadOpts('withLabels', 'labels')];
        this._store.dispatch(new FetchProject({ projectKey: key, opts }))
            .subscribe(null, () => this._router.navigate(['/home']));
    }

    updateFav() {
        this.loadingFav = true;
        this._store.dispatch(new UpdateFavoriteProject({ projectKey: this.project.key }))
            .pipe(finalize(() => {
                this.loadingFav = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('common_favorites_updated')));
    }
}
