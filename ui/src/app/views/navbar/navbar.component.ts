import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { FormControl } from '@angular/forms';
import { NavigationEnd, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Broadcast } from 'app/model/broadcast.model';
import { NavbarProjectData, NavbarRecentData, NavbarSearchItem } from 'app/model/navbar.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { ApplicationStore } from 'app/service/application/application.store';
import { BroadcastStore } from 'app/service/broadcast/broadcast.store';
import { LanguageStore } from 'app/service/language/language.store';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { RouterService } from 'app/service/router/router.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkflowStore } from 'app/service/workflow/workflow.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { SignoutCurrentUser } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { List } from 'immutable';
import { Subscription } from 'rxjs';
import { filter } from 'rxjs/operators';

@Component({
    selector: 'app-navbar',
    templateUrl: './navbar.html',
    styleUrls: ['./navbar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class NavbarComponent implements OnInit {
    // List of projects in the nav bar
    navProjects: Array<NavbarProjectData> = [];
    listFavs: Array<NavbarProjectData> = [];
    navRecentApp: List<Application>;
    navRecentWorkflows: List<NavbarRecentData>;
    searchItems: Array<NavbarSearchItem> = [];
    recentItems: Array<NavbarSearchItem> = [];
    items: Array<NavbarSearchItem> = [];
    broadcasts: Array<Broadcast> = new Array<Broadcast>();
    recentBroadcastsToDisplay: Array<Broadcast> = new Array<Broadcast>();
    previousBroadcastsToDisplay: Array<Broadcast> = new Array<Broadcast>();
    loading = true;
    listWorkflows: List<NavbarRecentData>;
    currentCountry: string;
    langSubscription: Subscription;
    navbarSubscription: Subscription;
    userSubscription: Subscription;
    broadcastSubscription: Subscription;
    currentRoute: {};
    recentView = true;
    currentUser: AuthentifiedUser;
    themeSubscription: Subscription;
    themeSwitch = new FormControl();

    constructor(
        private _navbarService: NavbarService,
        private _store: Store,
        private _appStore: ApplicationStore,
        private _workflowStore: WorkflowStore,
        private _broadcastStore: BroadcastStore,
        private _router: Router,
        private _language: LanguageStore,
        private _theme: ThemeStore,
        private _routerService: RouterService,
        private _cd: ChangeDetectorRef
    ) {
        this.userSubscription = this._store.select(AuthenticationState.user).subscribe(u => {
            this.currentUser = u;
            this._cd.markForCheck();
        });

        this.langSubscription = this._language.get().subscribe(l => {
            this.currentCountry = l;
            this._cd.markForCheck();
        });

        this.themeSubscription = this._theme.get().subscribe(t => {
            this.themeSwitch.setValue(t === 'night');
            this._cd.markForCheck();
        });

        this._router.events.pipe(
            filter(e => e instanceof NavigationEnd),
        ).forEach(() => {
            this.currentRoute = this._routerService.getRouteParams({}, this._router.routerState.root);
        });
    }

    changeCountry() {
        this._language.set(this.currentCountry);
    }

    changeTheme() {
        let darkActive = !!this.themeSwitch.value;
        this._theme.set(darkActive ? 'night' : 'light');
    }

    ngOnInit() {
        // Listen list of nav project
        this._store.selectOnce(AuthenticationState.user).subscribe(user => {
            if (user) {
                this.getData();
            }
        });

        // Listen change on recent app viewed
        this._appStore.getRecentApplications().subscribe(apps => {
            if (apps) {
                this.navRecentApp = apps;
                this.recentItems = this.recentItems
                    .filter((i) => i.type !== 'application')
                    .concat(
                        apps.toArray().map((app) => ({
                            type: 'application',
                            value: app.project_key + '/' + app.name,
                            title: app.name,
                            projectKey: app.project_key,
                            favorite: false
                        }))
                    );
                this.items = this.recentItems;
                this._cd.detectChanges();
            }
        });

        // Listen change on recent workflows viewed
        this._workflowStore.getRecentWorkflows().subscribe(workflows => {
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
                this._cd.detectChanges();
            }
        });
    }

    searchEvent(event) {
        if (!event || !event.target || !event.target.value) {
            this.items = this.recentItems;
        } else {
            let value = event.target.value;
            this.items = this.searchItems;
            event.target.value = value;
        }
    }

    /**
     * Listen change on project list.
     */
    getData(): void {
        this.navbarSubscription = this._navbarService.getData().subscribe(data => {
            if (Array.isArray(data) && data.length > 0) {
                this.navProjects = data;
                this.searchItems = new Array<NavbarSearchItem>();
                let favProj = [];
                this.listFavs = data.filter((p) => {
                    if (p.favorite && p.type !== 'workflow') {
                        if (p.type === 'project' && favProj.indexOf(p.key) === -1) {
                            favProj.push(p.key);
                            return true;
                        }
                        return false
                    }
                    return p.favorite;
                }).slice(0, 7);

                this.navProjects.forEach(p => {
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
                        case 'application':
                            this.searchItems.push({
                                value: p.key + '/' + p.application_name,
                                title: p.application_name,
                                type: 'application',
                                projectKey: p.key,
                                favorite: false
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

        this.broadcastSubscription = this._broadcastStore.getBroadcasts()
            .subscribe((broadcasts) => {
                let broadcastsToRead = broadcasts.valueSeq().toArray().filter(br => !br.read && !br.archived);
                let previousBroadcasts = broadcasts.valueSeq().toArray().filter(br => br.read && !br.archived);
                this.recentBroadcastsToDisplay = broadcastsToRead
                    .sort((a, b) => (new Date(b.updated)).getTime() - (new Date(a.updated)).getTime()).slice(0, 4);
                this.previousBroadcastsToDisplay = previousBroadcasts
                    .sort((a, b) => (new Date(b.updated)).getTime() - (new Date(a.updated)).getTime()).slice(0, 4);
                this.broadcasts = broadcastsToRead
                    .sort((a, b) => (new Date(b.updated)).getTime() - (new Date(a.updated)).getTime());
                this._cd.markForCheck();
            });
    }

    navigateToResult(result: NavbarSearchItem) {
        if (!result) {
            return;
        }
        switch (result.type) {
            case 'workflow':
                this.navigateToWorkflow(result.projectKey, result.value.split('/', 2)[1]);
                break;
            case 'application':
                this.navigateToApplication(result.projectKey, result.value.split('/', 2)[1]);
                break;
            default:
                this.navigateToProject(result.projectKey);
        }
    }

    searchItem(list: Array<NavbarSearchItem>, query: string): boolean | Array<NavbarSearchItem> {
        let queryLowerCase = query.toLowerCase();
        let found: Array<NavbarSearchItem> = [];
        for (let elt of list) {
            if (query === elt.projectKey) {
                found.push(elt);
            } else if (elt.title && elt.title.toLowerCase().indexOf(queryLowerCase) !== -1) {
                found.push(elt);
            }
        }
        return found;
    }

    /**
     * Navigate to the selected project.
     * @param key Project unique key get by the event
     */
    navigateToProject(key): void {
        this._router.navigate(['project/' + key]);
    }

    /**
     * Navigate to the selected application.
     */
    navigateToApplication(key: string, appName: string): void {
        this._router.navigate(['project', key, 'application', appName]);
    }

    /**
     * Navigate to the selected application.
     */
    navigateToWorkflow(key: string, workflowName: string): void {
        this._router.navigate(['project', key, 'workflow', workflowName]);
    }

    markAsRead(event: Event, id: number) {
        event.stopPropagation();
        this._broadcastStore.markAsRead(id)
            .subscribe();
    }

    clickLogout(): void {
        this._store.dispatch(new SignoutCurrentUser()).subscribe(
            () => { this._router.navigate(['/auth/signin']); }
        );
    }
}
