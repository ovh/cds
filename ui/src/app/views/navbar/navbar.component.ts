import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { NavigationEnd, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { APIConfig } from 'app/model/config.service';
import { Help } from 'app/model/help.model';
import { NavbarProjectData, NavbarRecentData, NavbarSearchItem } from 'app/model/navbar.model';
import { Project } from 'app/model/project.model';
import { AuthSummary } from 'app/model/user.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { RouterService } from 'app/service/router/router.service';
import { ProjectStore } from 'app/service/services.module';
import { WorkflowStore } from 'app/service/workflow/workflow.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { SignoutCurrentUser } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { ConfigState } from 'app/store/config.state';
import { HelpState } from 'app/store/help.state';
import { PreferencesState } from 'app/store/preferences.state';
import { List } from 'immutable';
import { Subscription, lastValueFrom } from 'rxjs';
import { filter } from 'rxjs/operators';
import * as actionPreferences from 'app/store/preferences.action';
import { ProjectService } from 'app/service/project/project.service';
import { NzMessageService } from 'ng-zorro-antd/message';
import { ErrorUtils } from 'app/shared/error.utils';
import { V2ProjectService } from 'app/service/projectv2/project.service';

@Component({
    selector: 'app-navbar',
    templateUrl: './navbar.html',
    styleUrls: ['./navbar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class NavbarComponent implements OnInit, OnDestroy {
    listFavs: Array<NavbarProjectData> = [];
    navRecentProjects: List<Project>;
    navRecentWorkflows: List<NavbarRecentData>;
    searchItems: Array<NavbarSearchItem> = [];
    recentItems: Array<NavbarSearchItem> = [];
    items: Array<NavbarSearchItem> = [];
    help: Help = new Help();
    loading = true;
    listWorkflows: List<NavbarRecentData>;
    navbarSubscription: Subscription;
    authSubscription: Subscription;
    configSubscription: Subscription;
    recentView = true;
    currentAuthSummary: AuthSummary;
    themeSubscription: Subscription;
    darkActive: boolean;
    searchValue: string;
    searchProjects: Array<NavbarSearchItem>;
    searchApplications: Array<NavbarSearchItem>;
    searchWorkflows: Array<NavbarSearchItem>;
    isSearch = false;
    containsResult = false;
    projectsSubscription: Subscription;
    workflowsSubscription: Subscription;
    showNotif = false;
    apiConfig: APIConfig;
    selectedProjectKey: string;
    projectSubscription: Subscription;
    projects: Array<Project> = [];

    constructor(
        private _navbarService: NavbarService,
        private _store: Store,
        private _projectStore: ProjectStore,
        private _workflowStore: WorkflowStore,
        private _router: Router,
        private _routerService: RouterService,
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _messageService: NzMessageService,
        private _v2ProjectService: V2ProjectService
    ) {
        this.authSubscription = this._store.select(AuthenticationState.summary).subscribe(s => {
            this.currentAuthSummary = s;
            this._cd.markForCheck();
        });

        this.configSubscription = this._store.select(ConfigState.api).subscribe(c => {
            this.apiConfig = c;
            this._cd.markForCheck();
        });

        this.themeSubscription = this._store.select(PreferencesState.theme).subscribe(t => {
            this.darkActive = t === 'night';
            this._cd.markForCheck();
        });

        this._store.select(HelpState.last)
            .pipe(
                filter((help) => help != null),
            )
            .subscribe(help => {
                this.help = help;
                this._cd.markForCheck();
            });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    changeTheme() {
        this.darkActive = !this.darkActive;
        this._cd.markForCheck();
        this._store.dispatch(new actionPreferences.SetTheme({ theme: this.darkActive ? 'night' : 'light' }));
    }

    ngOnInit() {
        // Listen list of nav project
        this._store.select(AuthenticationState.summary).subscribe(s => {
            if (s) {
                this.getData();
                this.loadProjects();
            }
        });

        // Listen change on recent projects viewed
        this.projectsSubscription = this._projectStore.getRecentProjects().subscribe(projects => {
            if (projects) {
                this.recentItems = projects.toArray().map((prj) => ({
                    type: 'project',
                    value: prj.project_key,
                    title: prj.name,
                    projectKey: prj.project_key
                })).concat(
                    this.recentItems.filter((i) => i.type !== 'project')
                );
                this.items = this.recentItems;
                this._cd.markForCheck();
            }
        });

        // Listen change on recent workflows viewed
        this.workflowsSubscription = this._workflowStore.getRecentWorkflows().subscribe(workflows => {
            if (workflows) {
                this.navRecentWorkflows = workflows;
                this.listWorkflows = workflows;
                this.recentItems = workflows.toArray().map((wf) => ({
                    type: 'workflow',
                    value: wf.project_key + '/' + wf.name,
                    title: wf.name,
                    projectKey: wf.project_key
                })).concat(
                    this.recentItems.filter((i) => i.type !== 'workflow')
                );
                this.items = this.recentItems;
                this._cd.markForCheck();
            }
        });

        this._router.events.pipe(
            filter(e => e instanceof NavigationEnd),
        ).forEach(() => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            this.selectedProjectKey = params['key'] ?? null;
            this._cd.markForCheck();
        });
    }

    search() {
        this.searchProjects = new Array<NavbarSearchItem>();
        this.searchApplications = new Array<NavbarSearchItem>();
        this.searchWorkflows = new Array<NavbarSearchItem>();

        if (this.searchValue && this.searchValue !== '') {
            this.isSearch = true;
            this.processSearch(this.searchItems);
            return;
        }

        // no search, display recentItems
        this.isSearch = false;
        this.processSearch(this.recentItems);
    }

    processSearch(items: Array<NavbarSearchItem>) {
        let searchPrjFull = false;
        let searchAppFull = false;
        let searchWfFull = false;

        let projectKey = '';
        let isProjectOnly = false;
        let containsProject = false;
        let firstPart = '';
        let secondPart = '';
        this.containsResult = false;

        if (this.searchValue && this.searchValue !== '') {
            isProjectOnly = this.searchValue.endsWith('/');
            containsProject = this.searchValue.includes('/');
            if (containsProject) {
                // FIRSTPART/SECONDPART
                firstPart = this.searchValue.substring(0, this.searchValue.indexOf('/')).toLowerCase();
                secondPart = this.searchValue.substring(this.searchValue.indexOf('/') + 1, this.searchValue.length).toLowerCase();
            }

            // if the search contains a project, get the current projectKey
            if (containsProject) {
                for (let index = 0; index < items.length; index++) {
                    const element = items[index];
                    if (element.type !== 'project') {
                        continue;
                    }
                    if (isProjectOnly) { // search end with '/'
                        if (element.title.toLowerCase() + '/' === this.searchValue.toLowerCase() ||
                            element.projectKey.toLowerCase() + '/' === this.searchValue.toLowerCase()) {
                            projectKey = element.projectKey;
                            break;
                        }
                    } else if ((element.title.toLowerCase() === firstPart) ||
                        (element.projectKey.toLowerCase() === firstPart)) {
                        projectKey = element.projectKey;
                        break;
                    }
                }
            }
        }

        for (let index = 0; index < items.length; index++) {
            const element = items[index];
            let toadd = false;
            if (!this.searchValue || this.searchValue === '') { // recent view
                toadd = true;
            } else if (isProjectOnly) {
                if (element.projectKey === projectKey) {
                    toadd = true;
                }
            } else if (containsProject) {
                // add project of firstpart/secondpart
                if (element.projectKey === projectKey && element.type === 'project') {
                    toadd = true;
                }

                if ((element.projectKey === projectKey) &&
                    element.title.toLowerCase().includes(secondPart)) {
                    toadd = true;
                }
            } else {
                // if search is not in projectKey and not in title, skip this item
                if (element.projectKey.toLowerCase().includes(this.searchValue.toLowerCase()) ||
                    element.title.toLowerCase().includes(this.searchValue.toLowerCase())) {
                    toadd = true;
                }
            }

            if (!toadd) {
                continue;
            }
            switch (element.type) {
                case 'project':
                    if (this.searchProjects.length < 10) {
                        this.searchProjects.push(element);
                        this.containsResult = true;
                    } else {
                        searchPrjFull = true;
                    }
                    break;
                case 'workflow':
                    if (this.searchWorkflows.length < 10) {
                        this.searchWorkflows.push(element);
                        this.containsResult = true;
                    } else {
                        searchWfFull = true;
                    }
                    break;
            }
            if (searchPrjFull && searchWfFull && searchAppFull) {
                break;
            }
        }
    }

    async getData() {
        this._navbarService.refreshData();
        this.navbarSubscription = this._navbarService.getObservable().subscribe(data => {
            if (Array.isArray(data) && data.length > 0) {
                this.searchItems = new Array<NavbarSearchItem>();
                let favProj = [];
                this.listFavs = data.filter((p) => {
                    if (p.favorite && p.type !== 'workflow') {
                        if (p.type === 'project' && favProj.indexOf(p.key) === -1) {
                            favProj.push(p.key);
                            return true;
                        }
                        return false;
                    }
                    return p.favorite;
                }).slice(0, 7);

                data.forEach(p => {
                    switch (p.type) {
                        case 'workflow':
                            this.searchItems.push({
                                value: p.key + '/' + p.workflow_name,
                                title: p.workflow_name,
                                type: 'workflow',
                                projectKey: p.key,
                                favorite: p.favorite
                            });
                            break;
                        default:
                            this.searchItems.push({
                                value: p.key,
                                title: p.name,
                                type: 'project',
                                projectKey: p.key,
                                favorite: p.favorite
                            });
                    }
                });
            }
            this.loading = false;
            this._cd.markForCheck();
        });
    }

    async loadProjects() {
        try {
            const projects = await lastValueFrom(this._projectService.getProjects());
            const projectsv2 = await lastValueFrom(this._v2ProjectService.getAll());
            this.projects = [].concat(projects)
                .concat(projectsv2.filter(pv2 => projects.findIndex(p => p.key === pv2.key) === -1));
            this.projects.sort((a, b) => { return a.name < b.name ? -1 : 1; })
            this._cd.markForCheck();
        } catch (e: any) {
            this._messageService.error(`Unable to load projects: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    navigateToProject(key): void {
        this._router.navigate(['project/' + key]);
    }

    navigateToWorkflow(key: string, workflowName: string): void {
        this._router.navigate(['project', key, 'workflow', workflowName]);
    }

    clickLogout(): void {
        this._store.dispatch(new SignoutCurrentUser()).subscribe(
            () => {
                this._router.navigate(['/auth/signin']);
            }
        );
    }

    mfaLogin(): void {
        const consumerType = this.currentAuthSummary.consumer.type;
        this._store.dispatch(new SignoutCurrentUser()).subscribe(
            () => {
                this._router.navigate([`/auth/ask-signin/${consumerType}`], {
                    queryParams: {
                        redirect_uri: this._router.url,
                        require_mfa: true
                    }
                });
            }
        );
    }
}
