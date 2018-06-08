import {AfterViewInit, Component, OnInit, ChangeDetectorRef} from '@angular/core';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {NavbarService} from '../../service/navbar/navbar.service';
import {ApplicationStore} from '../../service/application/application.store';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {BroadcastStore} from '../../service/broadcast/broadcast.store';
import {Application} from '../../model/application.model';
import {Broadcast} from '../../model/broadcast.model';
import {User} from '../../model/user.model';
import {NavigationEnd, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {List} from 'immutable';
import {LanguageStore} from '../../service/language/language.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {RouterService} from '../../service/router/router.service';
import {NavbarRecentData} from '../../model/navbar.model';
import {filter} from 'rxjs/operators';
import {NavbarSearchItem, NavbarProjectData} from 'app/model/navbar.model';

@Component({
    selector: 'app-navbar',
    templateUrl: './navbar.html',
    styleUrls: ['./navbar.scss']
})
@AutoUnsubscribe()
export class NavbarComponent implements OnInit, AfterViewInit {

    // flag to indicate that the component is ready to use
    public ready = false;

    // List of projects in the nav bar
    navProjects: Array<NavbarProjectData> = [];
    listFavs: Array<NavbarProjectData> = [];
    navRecentApp: List<Application>;
    navRecentWorkflows: List<NavbarRecentData>;
    searchItems: Array<NavbarSearchItem> = [];
    recentItems: Array<NavbarSearchItem> = [];
    items: Array<NavbarSearchItem> = [];
    broadcasts: Array<Broadcast> = [];
    recentBroadcastsToDisplay: Array<Broadcast> = [];
    previousBroadcastsToDisplay: Array<Broadcast> = [];
    loading = true;

    listWorkflows: List<NavbarRecentData>;

    currentCountry: string;
    langSubscription: Subscription;
    navbarSubscription: Subscription;
    userSubscription: Subscription;
    broadcastSubscription: Subscription;

    currentRoute: {};
    recentView = true;


    public currentUser: User;

    constructor(private _navbarService: NavbarService,
                private _authStore: AuthentificationStore,
                private _appStore: ApplicationStore,
                private _workflowStore: WorkflowStore,
                private _broadcastStore: BroadcastStore,
                private _router: Router, private _language: LanguageStore, private _routerService: RouterService,
                private _translate: TranslateService,
                private _authentificationStore: AuthentificationStore,
                private _cd: ChangeDetectorRef) {
        this.userSubscription = this._authentificationStore.getUserlst().subscribe(u => {
            this.currentUser = u;
        });

        this.langSubscription = this._language.get().subscribe(l => {
            this.currentCountry = l;
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

    ngAfterViewInit() {
        this._translate.get('navbar_projects_placeholder').subscribe(() => {
            this.ready = true;
        });
    }

    ngOnInit() {
        // Listen list of nav project
        this._authStore.getUserlst().subscribe(user => {
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
                        this.searchItems.push({value: p.key, title: p.name, type: 'project', projectKey: p.key, favorite: p.favorite});
                    }
                });
            }
            this.loading = false;
        });

        this.broadcastSubscription = this._broadcastStore.getBroadcasts()
            .subscribe((broadcasts) => {
                let broadcastsToRead = broadcasts.filter((br) => !br.read && !br.archived);
                let previousBroadcasts = broadcasts.filter((br) => br.read && !br.archived);
                this.recentBroadcastsToDisplay = broadcastsToRead.slice(0, 4);
                this.previousBroadcastsToDisplay = previousBroadcasts.slice(0, 4);
                this.broadcasts = broadcastsToRead;
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

    searchItem(list: Array<NavbarSearchItem>, query: string): boolean|Array<NavbarSearchItem> {
      let found: Array<NavbarSearchItem> = [];
      for (let elt of list) {
        if (query === elt.projectKey) {
          found.push(elt);
        } else if (elt.title.toLowerCase().indexOf(query.toLowerCase()) !== -1) {
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

    getWarningParams(): {} {
        return this.currentRoute;
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

    goToBroadcast(id: number): void {
        this._router.navigate(['broadcast', id]);
    }

    markAsRead(event: Event, id: number) {
        event.stopPropagation();
        this._broadcastStore.markAsRead(id)
            .subscribe();
    }
}
