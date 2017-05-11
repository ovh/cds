/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed, fakeAsync} from '@angular/core/testing';
import {AppModule} from './app.module';
import {AppComponent} from './app.component';

import {MockBackend} from '@angular/http/testing';
import { Http, RequestOptions, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {AuthentificationStore} from './service/auth/authentification.store';
import {User} from './model/user.model';
import {SharedModule} from './shared/shared.module';
import {ProjectStore} from './service/project/project.store';
import {ApplicationStore} from './service/application/application.store';
import {ProjectService} from './service/project/project.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Router, RouterModule} from '@angular/router';
import {ToastService} from './shared/toast/ToastService';
import {HttpService} from './service/http-service.service';
import {APP_BASE_HREF} from '@angular/common';
import {Observable} from 'rxjs/Observable';
import {ApplicationService} from './service/application/application.service';
import {PipelineService} from './service/pipeline/pipeline.service';
import {PipelineStore} from './service/pipeline/pipeline.store';
import {LastModification, ProjectLastUpdates} from './model/lastupdate.model';
import {AppService} from './app.service';
import {RouterTestingModule} from '@angular/router/testing';

describe('App: CDS', () => {

    let injector: Injector;
    let backend: MockBackend;
    let authStore: AuthentificationStore;
    let projectStore: ProjectStore;
    let applicationStore: ApplicationStore;
    let pipelineStore: PipelineStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                MockBackend,
                {
                    provide: Http,
                    useFactory: (backendParam: MockBackend,
                                 defaultOptions: RequestOptions,
                                 toast: ToastService,
                                 authStore2: AuthentificationStore,
                                 router: Router) =>
                        new HttpService(backendParam, defaultOptions, toast, authStore2, router),
                    deps: [MockBackend, RequestOptions, ToastService, AuthentificationStore]
                },
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: ProjectService, useClass: MockProjectService },
                { provide: ApplicationService, useClass: MockApplicationService},
                { provide: PipelineService, useClass: MockPipelineService},
                { provide: ActivatedRoute, useClass: MockActivatedRoutes}
            ],
            imports : [
                AppModule,
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

        injector = getTestBed();
        backend = injector.get(MockBackend);
        authStore = injector.get(AuthentificationStore);
        projectStore = injector.get(ProjectStore);
        applicationStore = injector.get(ApplicationStore);
        pipelineStore = injector.get(PipelineStore);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        authStore = undefined;
        projectStore = undefined;
        applicationStore = undefined;
        pipelineStore = undefined;
    });


    it('should create the app', async( () => {
        let fixture = TestBed.createComponent(AppComponent);
        let app = fixture.debugElement.componentInstance;
        expect(app).toBeTruthy();
    }));


    it('should render a navbar', async(() => {
        let fixture = TestBed.createComponent(AppComponent);
        let compiled = fixture.debugElement.nativeElement;

        expect(fixture.componentInstance.isConnected).toBeFalsy('IsConnected flag must be false');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must not have the css class "connected"');

        fixture.componentInstance.ngOnInit();
        authStore.addUser(new User(), false);

        expect(fixture.componentInstance.isConnected).toBeTruthy('IsConnected flag must be true');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must have connected css class');
    }));

    // FIXME CACHE NOT INITIALIZE
    it('should update cache', fakeAsync(() => {
        // Create cache
        projectStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        projectStore.getProjects('key2').first().subscribe(() => {}).unsubscribe();

        applicationStore.getApplications('key1', 'app1').first().subscribe(() => {}).unsubscribe();
        applicationStore.getApplications('key1', 'app2').first().subscribe(() => {}).unsubscribe();
        applicationStore.getApplications('key2', 'app3').first().subscribe(() => {}).unsubscribe();

        pipelineStore.getPipelines('key1', 'pip1').first().subscribe(() => {}).unsubscribe();
        pipelineStore.getPipelines('key1', 'pip2').first().subscribe(() => {}).unsubscribe();
        pipelineStore.getPipelines('key2', 'pip3').first().subscribe(() => {}).unsubscribe();


        let lastUpdateData = new Array<ProjectLastUpdates>();

        // Proj to keep
        let prj1 = new ProjectLastUpdates();
        prj1.name = 'key1';
        prj1.last_modified = 1497169222;
        prj1.username = 'fooApp';
        prj1.applications = new Array<LastModification>();
        prj1.pipelines = new Array<LastModification>();

        // App not modified
        let app1 = new LastModification();
        app1.last_modified = 1494490822;
        app1.name = 'app1';

        // Modified to keep
        let app2 = new LastModification();
        app2.last_modified = 1497169222;
        app2.username = 'fooApp';
        app2.name = 'app2';

        prj1.applications.push(app1);
        prj1.applications.push(app2);

        // Pip not updated
        let pip1 = new LastModification();
        pip1.last_modified = 1494490822;
        pip1.name = 'pip1';

        // Pip to keep
        let pip2 = new LastModification();
        pip2.last_modified = 1497169222;
        pip2.name = 'pip2';
        pip2.username = 'fooApp';

        prj1.pipelines.push(pip1, pip2);

        // Proj to delete
        let prj2 = new ProjectLastUpdates();
        prj2.name = 'key2';
        prj2.last_modified = 1497169222;
        prj2.applications = new Array<LastModification>();
        prj2.pipelines = new Array<LastModification>();

        // App to delete
        let app3 = new LastModification();
        app3.name = 'app3';
        app3.last_modified = 1497169222;
        app3.username = 'bar';

        prj2.applications.push(app3);

        let pip3 = new LastModification();
        pip3.name = 'pip3';
        pip3.last_modified = 1497169222;
        pip3.username = 'bar';

        prj2.pipelines.push(pip3);

        lastUpdateData.push(prj1, prj2);

        let AuthStore = injector.get(AuthentificationStore);
        let user = new User();
        user.username = 'fooApp';
        AuthStore.addUser(user, true);

        let appService = injector.get(AppService);

        appService.updateCache(lastUpdateData);

        // Check project result

        let check = false;
        projectStore.getProjects().subscribe(projs => {
            check = true;
            expect(projs.size).toBe(1, 'Must have just 1 project');
            expect(projs.get('key1')).toBeTruthy('project key1 must be here');
            expect(projs.get('key1').last_modified).toBe('2017-06-11T10:20:22.874779+02:00', 'project key1 have to be up to date');
        }).unsubscribe();
        expect(check).toBe(true);

        // Check application result
        let checkApp = false;
        applicationStore.getApplications('key').subscribe(apps => {
            checkApp = true;
            expect(apps.size).toBe(2, 'Must have 2 applications in cache');
            expect(apps.get('key1-app1')).toBeTruthy('app1 must be here');
            expect(apps.get('key1-app1').last_modified).toBe('2017-05-11T10:20:22.874779+02:00', 'No change on app1');
            expect(apps.get('key1-app2')).toBeTruthy('app2 must be here');
            expect(apps.get('key1-app2').last_modified).toBe('2017-06-11T10:20:22.874779+02:00', 'app2 have to be up to date');
        }).unsubscribe();
        expect(checkApp).toBe(true);

        // Check pipeline result
        let checkPip = false;
        pipelineStore.getPipelines('key').subscribe(pips => {
            checkPip = true;
            expect(pips.size).toBe(2, 'Must have 2 pipelines');
            expect(pips.get('key1-pip1')).toBeTruthy('pip1 must be here');
            expect(pips.get('key1-pip1').last_modified).toBe('2017-05-11T10:20:22.874779+02:00', 'no change on pip1');
            expect(pips.get('key1-pip2')).toBeTruthy('pip2 must be here');
            expect(pips.get('key1-pip2').last_modified).toBe('2017-06-11T10:20:22.874779+02:00', 'pip2 have to be up to date');
        }).unsubscribe();
        expect(checkPip).toBe(true);
    }));

});
class MockActivatedRoutes {

    snapshot: {
        params: {}
    };

    constructor() {
        this.snapshot = {
            params: {
                'key': 'key1',
                'appName': 'app2',
                'pipName': 'pip2'
            }
        };
    }
}

class MockProjectService extends ProjectService {
    callKEY1 = 0;
    getProject(key: string) {
        switch (key) {
            case 'key1':
                if (this.callKEY1 === 0) {
                    this.callKEY1++;
                    return Observable.of({ key: 'key1', name: 'project1', last_modified: '2017-05-11T10:20:22.874779+02:00'});
                } else {
                    return Observable.of({ key: 'key1', name: 'project1', last_modified: '2017-06-11T10:20:22.874779+02:00'});
                }
            case 'key2': return Observable.of({ key: 'key2', name: 'project2', last_modified: '2017-05-11T10:20:22.874779+02:00'});
        }

    }
}

class MockApplicationService extends ApplicationService {
    callAPP2 = 0;

    getApplication(key: string, appName: string) {
        if (key === 'key1') {
            if (appName === 'app1') {
                return Observable.of({ name: 'app1', last_modified: '2017-05-11T10:20:22.874779+02:00'});
            }
            if (appName === 'app2') {
                if (this.callAPP2 === 0) {
                    this.callAPP2++;
                    return Observable.of({ name: 'app2', last_modified: '2017-05-11T10:20:22.874779+02:00'});
                } else {
                    return Observable.of({ name: 'app2', last_modified: '2017-06-11T10:20:22.874779+02:00'});
                }

            }
        }
        if (key === 'key2') {
            if (appName === 'app3') {
                return Observable.of({ name: 'app3', last_modified: '2017-05-11T10:20:22.874779+02:00'});
            }
        }
    }
}

class MockPipelineService extends PipelineService {
    callPIP2 = 0;
    getPipeline(key: string, pipName: string) {
        if (key === 'key1') {
            if (pipName === 'pip1') {
                return Observable.of({ name: 'pip1', last_modified: '2017-05-11T10:20:22.874779+02:00'});
            }
            if (pipName === 'pip2') {
                if (this.callPIP2 === 0) {
                    this.callPIP2++;
                    return Observable.of({ name: 'pip2', last_modified: '2017-05-11T10:20:22.874779+02:00'});
                } else {
                    return Observable.of({ name: 'pip2', last_modified: '2017-06-11T10:20:22.874779+02:00'});
                }

            }
        }
        if (key === 'key2') {
            if (pipName === 'pip3') {
                return Observable.of({name: 'pip3', last_modified: '2017-05-11T10:20:22.874779+02:00'});
            }
        }
    }
}
