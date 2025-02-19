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
}
