import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { GroupPermission } from 'app/model/group.model';
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
            .pipe(filter((projState: ProjectStateModel) => projState && projState.project &&
                projState.project.key !== null && !projState.project.externalChange &&
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

                if (this.project.integrations) {
                    this.project.integrations.forEach(integ => {
                        if (!integ.model.default_config) {
                            return;
                        }
                        let keys = Object.keys(integ.model.default_config);
                        if (keys) {
                            keys.forEach(k => {
                                if (!integ.config) {
                                    integ.config = {};
                                }
                                if (!integ.config[k]) {
                                    integ.config[k] = integ.model.default_config[k];
                                }
                            });
                        }
                    });
                }


                if (!!this.project.organization) {
                    this.groupsOutsideOrganization = this.project.groups.filter(gp =>
                        gp.group.organization && gp.group.organization !== this.project.organization);
                }

                this._cd.markForCheck();
            });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.tabs = [<Tab>{
            translate: 'common_workflows',
            icon: 'share alternate',
            key: 'workflows',
            default: true,
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
            key: 'permissions',
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
        }]
        if (this.project?.permissions?.writable) {
            this.tabs.push(<Tab>{
                translate: 'common_advanced',
                icon: 'graduation',
                key: 'advanced'
            })
        }
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
