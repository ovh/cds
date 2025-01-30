import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { GroupPermission } from 'app/model/group.model';
import { LoadOpts, Project } from 'app/model/project.model';
import { HelpersService } from 'app/service/helpers/helpers.service';
import { ProjectStore, RouterService } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Tab } from 'app/shared/tabs/tabs.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchProject, UpdateFavoriteProject } from 'app/store/project.action';
import { ProjectState } from 'app/store/project.state';
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
export class ProjectShowComponent implements OnInit, OnDestroy, AfterViewInit {
    @ViewChild('tabPermissionTemplate') tabPermissionTemplate: TemplateRef<any>;

    project: Project;
    projectSubscriber: Subscription;
    groupsOutsideOrganization: Array<GroupPermission>;

    tabs: Array<Tab>;
    selectedTab: Tab;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;
    loadingFav = false;

    constructor(
        private _activatedRoute: ActivatedRoute,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _helpersService: HelpersService,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _projectStore: ProjectStore,
        private _routerService: RouterService,
        private _router: Router
    ) {
        this.projectSubscriber = this._store.select(ProjectState.projectSnapshot)
            .pipe(filter((p: Project) => p &&
                p.key !== null && !p.externalChange &&
                this._activatedRoute.snapshot.parent.params['key'] === p.key))
            .subscribe((p: Project) => {
                let proj = cloneDeep(p); // TODO: to delete when all will be in store, here it is usefull to skip readonly
                if (proj.labels) {
                    proj.labels = proj.labels.map((lbl) => {
                        lbl.font_color = this._helpersService.getBrightnessColor(lbl.color);
                        return lbl;
                    });
                }
                this.project = proj;
                this._projectStore.updateRecentProject(this.project);
                this.initTabs();

                if (!!this.project.organization) {
                    this.groupsOutsideOrganization = this.project.groups.filter(gp =>
                        gp.group.organization && gp.group.organization !== this.project.organization);
                }

                this._cd.markForCheck();
            });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.initTabs();

        this._activatedRoute.params.subscribe(_ => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            const key = params['key'];
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

        if (this._activatedRoute.snapshot && this._activatedRoute.snapshot.queryParams) {
            this.workflowName = this._activatedRoute.snapshot.queryParams['workflow'];
            this.workflowNum = this._activatedRoute.snapshot.queryParams['run'];
            this.workflowNodeRun = this._activatedRoute.snapshot.queryParams['node'];
            this.workflowPipeline = this._activatedRoute.snapshot.queryParams['wpipeline'];
        }
    }

    initTabs(): void {
        this.tabs = [<Tab>{
            title: 'Workflows',
            icon: 'share-alt',
            iconTheme: 'outline',
            key: 'workflows',
            default: true,
        }, <Tab>{
            title: 'Applications',
            icon: 'rocket',
            iconTheme: 'outline',
            key: 'applications'
        }, <Tab>{
            title: 'Pipelines',
            icon: 'apartment',
            key: 'pipelines'
        }, <Tab>{
            title: 'Environments',
            icon: 'environment',
            iconTheme: 'outline',
            key: 'environments'
        }, <Tab>{
            title: 'Variables',
            icon: 'font-colors',
            iconTheme: 'outline',
            key: 'variables'
        }, <Tab>{
            title: 'Permissions',
            key: 'permissions',
            iconTheme: 'outline',
            icon: 'user-switch',
        }];
    }

    ngAfterViewInit(): void {
        for (let i = 0; i < this.tabs.length; i++) {
            if (this.tabs[i].key === 'permissions') {
                // Change ref of permission tab to be detected in tab component
                let tab = cloneDeep(this.tabs[i]);
                tab.template = this.tabPermissionTemplate;
                this.tabs[i] = tab;

                // Change ref of this.tabs to trigger onPush change. ( so no need to trigger markForCheck )
                let newTabs = new Array<Tab>();
                this.tabs.forEach(d => {
                    newTabs.push(d);
                });
                this.tabs = newTabs;
            }
        }
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
        this._cd.markForCheck();
    }

    refreshDatas(key: string): void {
        let opts = [new LoadOpts('withLabels', 'labels')];
        this._store.dispatch(new FetchProject({ projectKey: key, opts }));
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
