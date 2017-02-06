import {Component, OnInit, AfterViewInit} from '@angular/core';
import {ProjectStore} from '../../service/project/project.store';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {Project} from '../../model/project.model';
import {ApplicationStore} from '../../service/application/application.store';
import {Application} from '../../model/application.model';
import {Router} from '@angular/router';
import {TranslateService} from 'ng2-translate';
import {List} from 'immutable';

@Component({
    selector: 'app-navbar',
    templateUrl: './navbar.html',
    styleUrls: ['./navbar.scss']
})
export class NavbarComponent implements OnInit, AfterViewInit {

    // flag to indicate that the component is ready to use
    private ready = false;

    // List of projects in the nav bar
    private navProjects: Project[];
    navRecentApp: List<Application>;
    selectedProject: Project = new Project();

    constructor(private _projectStore: ProjectStore,
                private _authStore: AuthentificationStore,
                private _appStore: ApplicationStore,
                private _router: Router,
                private _translate: TranslateService) {
    }

    ngAfterViewInit () {
        this._translate.get('navbar_projects_placeholder').subscribe( () => {
            this.ready = true;
        });
    }

    ngOnInit() {
        // Listen list of nav project
        this._authStore.getUserlst().subscribe( user => {
            if (user) {
                this.getProjects();
            }
        });

        // Listen change on recent app viewed
        this._appStore.getRecentApplications().subscribe( app => {
            if (app) {
                this.navRecentApp = app;
            }
        });
    }

    /**
     * Listen change on project list.
     */
    getProjects(): void {
        this._projectStore.getProjectsList().subscribe( projects => {
            if (projects) {
                this.navProjects = projects.toArray();
                if (this.selectedProject) {
                    let project = this.selectedProject;
                    this.navProjects.forEach(function (p) {
                        if (p.key === project.key) {
                            project = p;
                        }
                    });
                    this.selectedProject = project;
                }
            }
        });
    }

    /**
     * Navigate to the selected project.
     * @param key Project unique key get by the event
     */
    navigateToProject(key): void {
        this.selectedProject = this.navProjects.filter(p => p.key === key)[0];
        this._router.navigate(['/project/' + key]);
    }

    /**
     * Navigate to the selected application.
     * @param application Applicaiton to nagivate to
     */
    navigateToApplication(route): void {
        this._router.navigate([route]);
    }
}
