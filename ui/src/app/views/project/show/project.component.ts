import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { FetchProject, UpdateFavoriteProject } from 'app/store/project.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import * as immutable from 'immutable';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';
import { PermissionValue } from '../../../model/permission.model';
import { LoadOpts, Project } from '../../../model/project.model';
import { User } from '../../../model/user.model';
import { Warning } from '../../../model/warning.model';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { HelpersService } from '../../../service/helpers/helpers.service';
import { WarningStore } from '../../../service/warning/warning.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { Tab } from '../../../shared/tabs/tabs.component';
import { ToastService } from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss']
})
@AutoUnsubscribe()
export class ProjectShowComponent implements OnInit {
    currentUser: User;

    project: Project;
    projectSubscriber: Subscription;

    tabs: Array<Tab>;
    selectedTab: Tab;

    permissionEnum = PermissionValue;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;
    loadingFav = false;

    allWarnings: Array<Warning>;
    warnings: { [key: string]: Array<Warning> };
    warningsSub: Subscription;

    constructor(
        private _route: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _authentificationStore: AuthentificationStore,
        private _warningStore: WarningStore,
        private _helpersService: HelpersService,
        private store: Store
    ) {
        this.initWarnings();
        this.currentUser = this._authentificationStore.getUser();

        this.projectSubscriber = this.store.select(ProjectState)
            .pipe(filter((projState: ProjectStateModel) => {
                return projState && projState.project && projState.project.key !== null && !projState.project.externalChange &&
                    this._route.snapshot.params['key'] === projState.project.key;
            }))
            .subscribe((projState: ProjectStateModel) => {
                let proj = cloneDeep(projState.project); // TODO: to delete when all will be in store, here it is usefull to skip readonly
                if (proj.labels) {
                    proj.labels = proj.labels.map((lbl) => {
                        lbl.font_color = this._helpersService.getBrightnessColor(lbl.color);
                        return lbl;
                    });
                }
                this.project = proj;
            });
    }

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

    initWarnings() {
        this.warnings = {
            'workflows': new Array<Warning>(),
            'applications': new Array<Warning>(),
            'pipelines': new Array<Warning>(),
            'environments': new Array<Warning>(),
            'variables': new Array<Warning>(),
            'permissions': new Array<Warning>(),
            'keys': new Array<Warning>(),
            'advanced': new Array<Warning>(),
        };
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    refreshDatas(key: string): void {
        let opts = [
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withWorkflowNames', 'workflow_names'),
            new LoadOpts('withEnvironmentNames', 'environment_names'),
            new LoadOpts('withLabels', 'labels'),
        ];

        if (this.selectedTab) {
            switch (this.selectedTab.key) {
                case 'variables':
                    opts.push(new LoadOpts('withVariables', 'variables'));
                    break;
                case 'permissions':
                    opts.push(new LoadOpts('withEnvironments', 'environments'));
                    break;
            }
        }

        this.store.dispatch(new FetchProject({ projectKey: key, opts }))
            .subscribe(null, () => this._router.navigate(['/home']));

        this.warningsSub = this._warningStore.getProjectWarnings(key).subscribe(ws => {
            this.splitWarnings(ws.get(key));
        });
    }

    splitWarnings(warnings: immutable.Map<string, Warning>): void {
        if (warnings) {
            this.allWarnings = warnings.valueSeq().toArray().sort((a, b) => {
                return a.id - b.id;
            });
            this.initWarnings();
            this.allWarnings.forEach(v => {
                if (v.ignored) {
                    return;
                }
                if (v.application_name !== '') {
                    this.warnings['applications'].push(v);
                }
                if (v.pipeline_name !== '') {
                    this.warnings['pipelines'].push(v);
                }
                if (v.environment_name !== '') {
                    this.warnings['environments'].push(v);
                }
                if (v.workflow_name !== '') {
                    this.warnings['workflows'].push(v);
                }
                if (v.type.indexOf('_VARIABLE') !== -1) {
                    this.warnings['variables'].push(v);
                    return;
                }
                if (v.type.indexOf('_PERMISSION') !== -1) {
                    this.warnings['permissions'].push(v);
                    return;
                }
                if (v.type.indexOf('_KEY') !== -1) {
                    this.warnings['keys'].push(v);
                    return;
                }
                if (v.type.indexOf('PROJECT_VCS') !== -1) {
                    this.warnings['advanced'].push(v);
                    return;
                }
            });
        }
    }

    updateFav() {
        this.loadingFav = true;
        this.store.dispatch(new UpdateFavoriteProject({ projectKey: this.project.key }))
            .pipe(finalize(() => this.loadingFav = false))
            .subscribe(() => this._toast.success('', this._translate.instant('common_favorites_updated')));
    }
}
