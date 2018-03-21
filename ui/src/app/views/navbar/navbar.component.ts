import {AfterViewInit, Component, OnInit, ChangeDetectorRef} from '@angular/core';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {NavbarService} from '../../service/navbar/navbar.service';
import {ApplicationStore} from '../../service/application/application.store';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {Application} from '../../model/application.model';
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
import {NavbarData, NavbarSearchItem, NavbarProjectData} from 'app/model/navbar.model';

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
    navProjects: NavbarData;
    listProjects: Array<NavbarProjectData> = [];
    navRecentApp: List<Application>;
    navRecentWorkflows: List<NavbarRecentData>;
    searchItems: Array<NavbarSearchItem> = [];
    recentItems: Array<NavbarSearchItem> = [];
    items: Array<NavbarSearchItem> = [];

    listApplications: List<Application>;
    listWorkflows: List<NavbarRecentData>;

    currentCountry: string;
    langSubscrition: Subscription;

    currentRoute: {};

    userSubscription: Subscription;

    public currentUser: User;

    constructor(private _navbarService: NavbarService,
                private _authStore: AuthentificationStore,
                private _appStore: ApplicationStore,
                private _workflowStore: WorkflowStore,
                private _router: Router, private _language: LanguageStore, private _routerService: RouterService,
                private _translate: TranslateService,
                private _authentificationStore: AuthentificationStore,
                private _cd: ChangeDetectorRef) {
        this.userSubscription = this._authentificationStore.getUserlst().subscribe(u => {
            this.currentUser = u;
        });

        this.langSubscrition = this._language.get().subscribe(l => {
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
                this.listApplications = apps;
                this.recentItems = this.recentItems
                    .filter((i) => i.type !== 'application')
                    .concat(
                        apps.toArray().map((app) => ({
                            type: 'application',
                            value: app.project_key + '/' + app.name,
                            title: app.name,
                            projectKey: app.project_key
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
                this.recentItems = workflows.toArray()
                    .map((w) => ({
                        type: 'workflow',
                        value: w.project_key + '/' + w.name,
                        title: w.name,
                        projectKey: w.project_key
                    }))
                    .concat(this.recentItems.filter((i) => i.type !== 'workflow'));
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
        this._navbarService.getData().subscribe(data => {
            if (data.projects && data.projects.length > 0) {
                data.projects = data.projects.concat(data.projects).concat(data.projects).concat(data.projects).concat(data.projects);
                this.navProjects = data;
                this.listProjects = data.projects.slice(0, 10);
                this.searchItems = new Array<NavbarSearchItem>();

                this.navProjects.projects.forEach(p => {
                    this.searchItems.push({value: p.key, title: p.name, type: 'project', projectKey: p.key});
                    if (p.application_names && p.application_names.length > 0) {
                        p.application_names.forEach(a => {
                            this.searchItems.push({value: p.key + '/' + a, title: a, type: 'application', projectKey: p.key});
                        });
                    }
                    if (p.workflow_names && p.workflow_names.length > 0) {
                        p.workflow_names.forEach(w => {
                            this.searchItems.push({value: p.key + '/' + w, title: w, type: 'workflow', projectKey: p.key});
                        });
                    }
                });
            }
        });
    }

    navigateToResult(result: string) {
        let splittedSelection = result.split('/', 2);
        let project = this.navProjects.projects.find(p => p.key === splittedSelection[0]);

        if (splittedSelection.length === 1) {
            this.navigateToProject(project.key);
        } else if (splittedSelection.length === 2) {
            if (Array.isArray(project.workflow_names)) {
                let workflowFound = project.workflow_names.find(w => w === splittedSelection[1]);
                if (workflowFound) {
                    this.items = this.recentItems;
                    return this.navigateToWorkflow(project.key, workflowFound);
                }
            }

            if (Array.isArray(project.application_names)) {
                let appFound = project.application_names.find(a => a === splittedSelection[1]);
                if (appFound) {
                    this.items = this.recentItems;
                    return this.navigateToApplication(project.key, appFound);
                }
            }
        }
    }

    selectAllProjects(): void {
        this.listApplications = this.navRecentApp;
    }

    /**
     * Navigate to the selected project.
     * @param key Project unique key get by the event
     */
    navigateToProject(key): void {
        if (key === '#NOPROJECT#') {
            this.selectAllProjects();
            return;
        }

        let selectedProject = this.navProjects.projects.filter(p => {
            return p.key === key;
        })[0];
        let apps = selectedProject.application_names.map((a) => {
            let app = new Application();
            app.name = a;
            app.project_key = selectedProject.key;
            return app
        });
        this.listApplications = List(apps);
        this._router.navigate(['/project/' + key]);
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
}
