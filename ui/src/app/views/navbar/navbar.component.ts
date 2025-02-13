import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, NavigationEnd, Router } from '@angular/router';
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
import { Filter, InputFilterComponent, Suggestion } from 'app/shared/input/input-filter.component';
import { SearchService } from 'app/service/search.service';
import Debounce from 'app/shared/decorator/debounce';
import { SearchResult, SearchResultType } from 'app/model/search.model';

@Component({
    selector: 'app-navbar',
    templateUrl: './navbar.html',
    styleUrls: ['./navbar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class NavbarComponent implements OnInit, OnDestroy {
    @ViewChild('searchBar') searchBar: InputFilterComponent<NavbarSearchItem>;

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
        private _v2ProjectService: V2ProjectService,
        private _searchService: SearchService,
        private _activatedRoute: ActivatedRoute
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
                this.loadFilters();
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
            const res = await Promise.all([
                lastValueFrom(this._projectService.getProjects()),
                lastValueFrom(this._v2ProjectService.getAll())
            ]);
            this.projects = [].concat(res[0])
                .concat(res[1].filter(pv2 => res[0].findIndex(p => p.key === pv2.key) === -1));
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

    searchFilterText: string = '';
    searchFilters: Array<Filter> = [];
    searchSuggestions: Array<Suggestion<SearchResult>> = [];

    selectSuggestion(value: SearchResult): void {
        const splitted = value.id.split('/');
        switch (value.type) {
            case SearchResultType.Workflow:
                const project = splitted.shift();
                const workflow_path = splitted.join('/');
                this._router.navigate(['/project', project, 'run'], {
                    queryParams: {
                        workflow: workflow_path
                    }
                });
                return;
            case SearchResultType.WorkflowLegacy:
                this._router.navigate(['/project', splitted[0], 'workflow', splitted[1]]);
                return;
            case SearchResultType.Project:
                this._router.navigate(['/project', value.id]);
                return;
            default:
                return;
        }
    }

    generateResultLink(res: SearchResult): Array<string> {
        const splitted = res.id.split('/');
        switch (res.type) {
            case SearchResultType.Workflow:
                const project = splitted.shift();
                return ['/project', project, 'run'];
            case SearchResultType.WorkflowLegacy:
                return ['/project', splitted[0], 'workflow', splitted[1]];
            case SearchResultType.Project:
                return ['/project', res.id];
            default:
                return [];
        }
    }

    generateResulQueryParams(res: SearchResult, variant?: string): any {
        const splitted = res.id.split('/');
        switch (res.type) {
            case SearchResultType.Workflow:
                splitted.shift();
                const workflow_path = splitted.join('/');
                let params = { workflow: workflow_path };
                if (variant) {
                    params['ref'] = variant;
                }
                return params;
            default:
                return {};
        }
    }

    submitSearch(): void {
        let mFilters = {};
        this.searchFilterText.split(' ').forEach(f => {
            const s = f.split(':');
            if (s.length === 2 && s[1] !== '') {
                if (!mFilters[s[0]]) {
                    mFilters[s[0]] = [];
                }
                mFilters[s[0]].push(s[1]);
            } else if (s.length === 1) {
                mFilters['query'] = f;
            }
        });

        this._router.navigate(['/search'], {
            queryParams: { ...mFilters },
            replaceUrl: true
        });
    }

    searchChange(v: string) {
        this.searchFilterText = v;
        this.search();
    }

    @Debounce(300)
    async search() {
        this.loading = true;
        this._cd.markForCheck();

        let mFilters = {};
        this.searchFilterText.split(' ').forEach(f => {
            const s = f.split(':');
            if (s.length === 2) {
                if (!mFilters[s[0]]) {
                    mFilters[s[0]] = [];
                }
                mFilters[s[0]].push(decodeURI(s[1]));
            } else if (s.length === 1) {
                mFilters['query'] = f;
            }
        });

        try {
            const res = await lastValueFrom(this._searchService.search(mFilters, 0, 10));
            this.searchSuggestions = res.results.map(r => ({
                key: r.id,
                label: `${r.label} - ${r.id}`,
                data: r,
            }));
        } catch (e: any) {
            this._messageService.error(`Unable to search: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }

    async loadFilters() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            this.searchFilters = await lastValueFrom(this._searchService.getFilters());
        } catch (e) {
            this._messageService.error(`Unable to list search filters: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        this.loading = false;
        this._cd.markForCheck();
    }

    clickSuggestion(): void {
        this.searchBar.filterInputDirective.closePanel();
    }
}
