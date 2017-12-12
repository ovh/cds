import {AfterViewInit, Component, OnInit} from '@angular/core';
import {ProjectStore} from '../../service/project/project.store';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {NavbarService} from '../../service/navbar/navbar.service';
import {ApplicationStore} from '../../service/application/application.store';
import {Application} from '../../model/application.model';
import {User} from '../../model/user.model';
import {NavigationEnd, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {List} from 'immutable';
import {LanguageStore} from '../../service/language/language.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {RouterService} from '../../service/router/router.service';
import {WarningStore} from '../../service/warning/warning.store';
import {WarningUI} from '../../model/warning.model';
import {WarningService} from '../../service/warning/warning.service';
import {filter} from 'rxjs/operators';
import {NavbarData, NavbarProjectData} from 'app/model/navbar.model';

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
    navRecentApp: List<Application>;
    searchItems: Array<string>;

    listApplications: List<Application>;

    currentCountry: string;
    langSubscrition: Subscription;

    warnings: Map<string, WarningUI>;
    warningsCount: number;
    currentRoute: {};

    userSubscription: Subscription;
    warningSubscription: Subscription;

    public currentUser: User;

    constructor(private _navbarService: NavbarService,
                private _authStore: AuthentificationStore,
                private _appStore: ApplicationStore,
                private _router: Router, private _language: LanguageStore, private _routerService: RouterService,
                private _translate: TranslateService, private _warningStore: WarningStore,
                private _authentificationStore: AuthentificationStore, private _warningService: WarningService) {
        this.userSubscription = this._authentificationStore.getUserlst().subscribe(u => {
            this.currentUser = u;
        });

        this.langSubscrition = this._language.get().subscribe(l => {
            this.currentCountry = l;
        });

        this.warningSubscription = this._warningStore.getWarnings().subscribe(ws => {
            this.warnings = ws;
            this.warningsCount = this._warningService.calculateWarningCountForCurrentRoute(this.currentRoute, this.warnings);
        });

        this._router.events.pipe(
            filter(e => e instanceof NavigationEnd),
        ).forEach(() => {
            this.currentRoute = this._routerService.getRouteParams({}, this._router.routerState.root);
            this.warningsCount = this._warningService.calculateWarningCountForCurrentRoute(this.currentRoute, this.warnings);
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
            }
        });
    }

    /**
     * Listen change on project list.
     */
    getData(): void {
        this._navbarService.getData().subscribe(data => {
            if (data.projects && data.projects.length > 0) {
                this.navProjects = data;
                this.searchItems = new Array<string>();

                this.navProjects.projects.forEach(p => {
                    if (p.application_names && p.application_names.length > 0) {
                        p.application_names.forEach(a => {
                            this.searchItems.push(p.name + '/' + a);
                        })
                    }
                });
            }
        });
    }

    navigateToResult(result: string) {
        let splittedSelection = result.split('/', 2);
        let project = this.navProjects.projects.find(p => p.name === splittedSelection[0]);
        if (splittedSelection.length === 1) {
            this.navigateToProject(project.key);
        } else if (splittedSelection.length === 2) {
            this.navigateToApplication(project.key, project.application_names.find(a => a === splittedSelection[1]));
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
}
