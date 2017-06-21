import {AfterViewInit, Component, OnDestroy, OnInit} from '@angular/core';
import {ProjectStore} from '../../service/project/project.store';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {Project} from '../../model/project.model';
import {ApplicationStore} from '../../service/application/application.store';
import {Application} from '../../model/application.model';
import {User} from '../../model/user.model';
import {NavigationEnd, Router} from '@angular/router';
import {TranslateService} from 'ng2-translate';
import {List} from 'immutable';
import {LanguageStore} from '../../service/language/language.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {RouterService} from '../../service/router/router.service';
import {WarningStore} from '../../service/warning/warning.store';
import {WarningUI} from '../../model/warning.model';
import {WarningService} from '../../service/warning/warning.service';

@Component({
    selector: 'app-navbar',
    templateUrl: './navbar.html',
    styleUrls: ['./navbar.scss']
})
@AutoUnsubscribe()
export class NavbarComponent implements OnInit, OnDestroy, AfterViewInit {

    // flag to indicate that the component is ready to use
    public ready = false;

    // List of projects in the nav bar
    navProjects: List<Project>;
    navRecentApp: List<Application>;

    selectedProjectKey: string;
    selectedApplicationName: string;
    listApplications: Array<Application>;

    currentCountry: string;
    langSubscrition: Subscription;

    warnings: Map<string, WarningUI>;
    warningsCount: number;
    currentRoute: {};

    userSubscription: Subscription;
    warningSubscription: Subscription;

    public currentUser: User;

    constructor(private _projectStore: ProjectStore,
                private _authStore: AuthentificationStore,
                private _appStore: ApplicationStore,
                private _router: Router, private _language: LanguageStore, private _routerService: RouterService,
                private _translate: TranslateService, private _warningStore: WarningStore,
                private _authentificationStore: AuthentificationStore, private _warningService: WarningService) {
        this.selectedProjectKey = '#NOPROJECT#';
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

        this._router.events
            .filter(e => e instanceof NavigationEnd)
            .forEach(() => {
                this.currentRoute = this._routerService.getRouteParams({}, this._router.routerState.root);
                this.warningsCount = this._warningService.calculateWarningCountForCurrentRoute(this.currentRoute, this.warnings);
            });
    }

    changeCountry() {
        this._language.set(this.currentCountry);
    }

    ngOnDestroy() {
        if (this.langSubscrition) {
            this.langSubscrition.unsubscribe();
        }
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
                this.getProjects();
            }
        });

        // Listen change on recent app viewed
        this._appStore.getRecentApplications().subscribe(app => {
            if (app) {
                this.navRecentApp = app;
                this.listApplications = this.navRecentApp.toArray();
            }
        });
    }



    /**
     * Listen change on project list.
     */
    getProjects(): void {
        this._projectStore.getProjectsList().subscribe(projects => {
            if (projects.size > 0) {
                this.navProjects = projects;
            }
        });
    }

    selectAllProjects(): void {
        this.listApplications = this.navRecentApp.toArray();
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

        let selectedProject = this.navProjects.filter(p => {
            return p.key === key;
        }).toArray()[0];
        this.listApplications = selectedProject.applications;
        this._router.navigate(['/project/' + key]);
    }

    getWarningParams(): {} {
        return this.currentRoute;
    }

    /**
     * Navigate to the selected application.
     */
    navigateToApplication(route: string): void {
        if (route === '#NOAPP#') {
            return;
        }
        this.selectedApplicationName = '#NOAPP#';
        this._router.navigate([route]);
    }

    applicationKeyEvent(event: KeyboardEvent, a): void {
        if (event.key === 'Escape') {
            this.selectedProjectKey = '#NOPROJECT#';
            this.selectedApplicationName = '#NOAPP#';
            this.selectAllProjects();
        }
    }

    filterApplication(event: any): void {
        if (this.selectedProjectKey === '#NOPROJECT#') {
            if (this.navProjects) {
                this.listApplications = this.navProjects.toArray().reduce((allProj, proj) => {
                    let filteredApps = [];

                    if (proj.applications) {
                        filteredApps = proj.applications.filter(app => {
                            return app.name.toLocaleLowerCase().indexOf(event.toLowerCase()) !== -1;
                        });
                    }

                    return [...allProj, ...filteredApps];
                }, []);
            }
        }
    }
}
