/* tslint:disable:no-unused-variable */

import { APP_BASE_HREF } from '@angular/common';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { async, fakeAsync, TestBed } from '@angular/core/testing';
import { RouterModule } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { AppModule } from '../../app.module';
import { Application } from '../../model/application.model';
import { Project } from '../../model/project.model';
import { Variable } from '../../model/variable.model';
import { ApplicationService } from './application.service';
import { ApplicationStore } from './application.store';

describe('CDS: application Store', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: APP_BASE_HREF, useValue: '/'},
                {provide: ApplicationService, useClass: MockApplicationService}
            ],
            imports: [
                AppModule,
                RouterModule,
                HttpClientTestingModule
            ]
        });

    });

    it('should create and delete an Application', fakeAsync(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let app1 = new Application();
        app1.name = 'myApplication';

        let app2 = new Application();
        app2.name = 'myApplication2';


        // Create Get application
        let checkApplicationCreated = false;
        applicationStore.getApplicationResolver('key1', 'myApplication').subscribe(res => {
            expect(res.name).toBe('myApplication', 'Wrong application name');
            checkApplicationCreated = true;
        });

        expect(checkApplicationCreated).toBeTruthy();

        // check get application (get from cache)
        let checkedSingleApplication = false;
        applicationStore.getApplications('key1', 'myApplication').subscribe(apps => {
            expect(apps.get('key1' + '-myApplication').name).toBe('myApplication', 'Wrong application name. Must be myApplication');
            checkedSingleApplication = true;
        }).unsubscribe();
        expect(checkedSingleApplication).toBeTruthy('Need to get application myApplication');


        let checkedDeleteApp = false;
        applicationStore.deleteApplication('key1', 'myApplication2').subscribe(() => {
        });

        applicationStore.getApplications('key1', 'myApplication2').subscribe(() => {
            checkedDeleteApp = true;
        }).unsubscribe();
        expect(checkedDeleteApp).toBeTruthy();
    }));

    it('should update the application', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let application = new Application();
        application.name = 'myApplication';

        let applicationU = new Application();
        applicationU.name = 'myApplicationUpdate1';

        let projectKey = 'key1';


        // Create application
        let p = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, p.name).subscribe(() => {
        });

        // Update
        p.name = 'myApplicationUpdate1';
        applicationStore.updateApplication(projectKey, 'myApplication', applicationU).subscribe(() => {
        });
        // check get application
        let checkedApplication = false;
        applicationStore.getApplications(projectKey, 'myApplicationUpdate1').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplicationUpdate1').name)
                .toBe('myApplicationUpdate1', 'Wrong application name. Must be myApplicationUpdate1');
            checkedApplication = true;
        }).unsubscribe();
        expect(checkedApplication).toBeTruthy('Need to get application myApplicationUpdate1');
    }));

    it('should attach then Detach a repository', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let application = new Application();
        application.name = 'myApplication';
        application.last_modified = '123';

        let applicationC = new Application();
        applicationC.name = 'myApplicationUpdate1';
        applicationC.last_modified = '456';
        applicationC.vcs_server = 'repoman';
        applicationC.repository_fullname = 'myrepo';

        let applicationLast = new Application();
        applicationLast.name = 'myApplicationUpdate1';
        applicationLast.last_modified = '789';

        let projectKey = 'key1';

        // Create application
        let p = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, p.name).subscribe(() => {
        });

        applicationStore.connectRepository(projectKey, 'myApplication', 'repoman', 'myrepo').subscribe(() => {
        });

        // check get application
        let checkedAttached = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplication').repository_fullname).toBe('myrepo', 'Repo fullname must be set to "myrepo"');
            expect(apps.get(projectKey + '-myApplication').vcs_server)
                .toBe('repoman', 'Repo manager must be set to "repoman"');
            checkedAttached = true;
        }).unsubscribe();
        expect(checkedAttached).toBeTruthy('Need application to be updated');

        applicationStore.removeRepository(projectKey, 'myApplication', 'repoman').subscribe(() => {
        });

        // check get application
        let checkedDettach = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplication').repository_fullname).toBeFalsy('Repo fullname must not be set');
            expect(apps.get(projectKey + '-myApplication').repositories_manager).toBeFalsy('Repo manager must not be set');
            checkedDettach = true;
        }).unsubscribe();
        expect(checkedDettach).toBeTruthy('Need application to be updated');
    }));


    it('should add/update/delete a variable', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let app = new Application();
        app.name = 'myApplication';
        app.last_modified = '0';

        let appAddVar = new Application();
        appAddVar.name = 'myApplication';
        appAddVar.last_modified = '123';
        appAddVar.variables = new Array<Variable>();
        let v1 = new Variable();
        v1.name = 'foo';
        appAddVar.variables.push(v1);

        let appUpVar = new Application();
        appUpVar.name = 'myApplication';
        appUpVar.last_modified = '456';
        appUpVar.variables = new Array<Variable>();
        let v2 = new Variable();
        v2.name = 'bar';
        appUpVar.variables.push(v2);

        let appDelVar = new Application();
        appDelVar.name = 'myApplication';
        appDelVar.last_modified = '789';
        appDelVar.variables = new Array<Variable>();


        let proj: Project = new Project();
        proj.key = 'key1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(proj.key, a.name).subscribe(() => {
        });

        let v: Variable = new Variable();
        v.name = 'foo';


        applicationStore.addVariable(proj.key, a.name, v).subscribe(() => {
        });

        // check get application
        let checkedAddVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(1, 'A variable must have been added');
            expect(apps.get(proj.key + '-myApplication').variables[0].name).toBe('foo', 'Variable name must be foo');
            checkedAddVariable = true;
        }).unsubscribe();
        expect(checkedAddVariable).toBeTruthy('Need application to be updated');


        v.name = 'bar';
        applicationStore.updateVariable(proj.key, a.name, v).subscribe(() => {
        });

        // check get application
        let checkedUpdateVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(1, 'App must have 1 variables');
            expect(apps.get(proj.key + '-myApplication').variables[0].name).toBe('bar', 'Variable name must be bar');
            checkedUpdateVariable = true;
        }).unsubscribe();
        expect(checkedUpdateVariable).toBeTruthy('Need application to be updated');


        applicationStore.removeVariable(proj.key, a.name, v).subscribe(() => {
        });
        // check get application
        let checkedDeleteVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(0, 'App must have 0 variable');
            checkedDeleteVariable = true;
        }).unsubscribe();
        expect(checkedDeleteVariable).toBeTruthy('Need application to be updated');
    }));

    function createApplication(name: string): Application {
        let app: Application = new Application();
        app.name = name;
        return app;
    }

    class MockApplicationService {

        getApplication(key: string, appName: string, filter?: {branch: string, remote: string}): Observable<Application> {
            let app = new Application();
            app.name = appName;
            return Observable.of(app);
        }

        deleteApplication(key: string, appName: string): Observable<boolean> {
            return Observable.of(true);
        }

        updateApplication(key: string, oldName: string, appli: Application): Observable<Application> {
            return Observable.of(appli);
        }

        connectRepository(key: string, currentName: string, repoManName: string, repoFullname: string): Observable<Application> {
            let app = new Application();
            app.name = currentName;
            app.repository_fullname = repoFullname;
            app.vcs_server = repoManName;
            return Observable.of(app);
        }

        addVariable(key: string, appName: string, v: Variable): Observable<Application> {
            let app = new Application();
            app.name = appName;
            app.variables = new Array<Variable>();
            app.variables.push(v);
            return Observable.of(app);
        }

        updateVariable(key: string, appName: string, v: Variable): Observable<Application> {
            let app = new Application();
            app.name = appName;
            app.variables = new Array<Variable>();
            app.variables.push(v);
            return Observable.of(app);
        }

        removeVariable(key: string, appName: string, v: Variable): Observable<Application> {
            let app = new Application();
            app.name = appName;
            app.variables = new Array<Variable>();
            return Observable.of(app);
        }

        removeRepository(key: string, currentName: string, repoManName: string): Observable<Application> {
            let app = new Application();
            app.name = currentName;
            return Observable.of(app);
        }
    }
});
