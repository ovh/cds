/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed} from '@angular/core/testing';
import {AppModule} from './app.module';
import {AppComponent} from './app.component';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend, MockConnection} from '@angular/http/testing';
import {ConnectionBackend, Http, RequestOptions, ResponseOptions, XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {AuthentificationStore} from './service/auth/authentification.store';
import {User} from './model/user.model';
import {SharedModule} from './shared/shared.module';
import {TranslateService, TranslateParser} from 'ng2-translate';
import {AppService} from './app.service';
import {ProjectStore} from './service/project/project.store';
import {ApplicationStore} from './service/application/application.store';
import {PipelineStore} from './service/pipeline/pipeline.store';
import {ProjectService} from './service/project/project.service';
import {ApplicationService} from './service/application/application.service';
import {PipelineService} from './service/pipeline/pipeline.service';
import {LastModification, ProjectLastUpdates} from './model/lastupdate.model';
import {ActivatedRoute, ActivatedRouteSnapshot, Router} from '@angular/router';
import {Observable} from 'rxjs/Observable';
import {Project} from './model/project.model';
import {ToastService} from './shared/toast/ToastService';
import {HttpService} from './service/http-service.service';
import {APP_BASE_HREF} from '@angular/common';

describe('App: CDS', () => {

    let injector: Injector;
    let backend: MockBackend;
    let authStore: AuthentificationStore;

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
                TranslateService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                AppService,
                TranslateParser,
                ProjectStore, ProjectService,
                ApplicationStore, ApplicationService,
                PipelineStore, PipelineService,
                AuthentificationStore,
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
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        authStore = undefined;
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
    it('should update cache', async(() => {
        /*
        let call = 0;
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0:
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "myProject" }'})));
                    break;
                case 1:
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key2", "name": "myProject2" }'})));
                    break;
                case 2:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "myProject" }'})));
                    break;
                case 3:
                case 4:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ }'})));
                    break;
            }


        });
        // Mock Http

        backend.connections.subscribe(connection => {
            call++;
            console.log(call);
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({body: JSON.stringify({ "key": "key1", "name": "myProject",
                    "last_modified": 123})})));
                    break;
                case 2:
                    connection.mockRespond(new Response('{ "key" : "key2", "last_modified": 123}'));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name" : "app1", "last_modified": 123}'})));
                    break;
                case 4:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name" : "app2", "last_modified": 123}'})));
                    break;
                case 5:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name" : "app3", "last_modified": 123}'})));
                    break;
                case 6:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name" : "pip1", "last_modified": 123}'})));
                    break;
                case 7:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name" : "pip2", "last_modified": 123}'})));
                    break;
                case 8:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name" : "pip3", "last_modified": 123}'})));
                    break;
            }

        });

        let projectStore = injector.get(ProjectStore);
        let appStore = injector.get(ApplicationStore);
        let pipStore = injector.get(PipelineStore);

        // Create cache
        projectStore.getProjects('key1').subscribe(projs => {

        }).unsubscribe();
        projectStore.getProjects('key2').first().subscribe(() => {}).unsubscribe();

        appStore.getApplications('key1', 'app1').first().subscribe(() => {}).unsubscribe();
        appStore.getApplications('key1', 'app2').first().subscribe(() => {}).unsubscribe();
        appStore.getApplications('key2', 'app3').first().subscribe(() => {}).unsubscribe();

        pipStore.getPipelines('key1', 'pip1').first().subscribe(() => {}).unsubscribe();
        pipStore.getPipelines('key1', 'pip2').first().subscribe(() => {}).unsubscribe();
        pipStore.getPipelines('key2', 'pip3').first().subscribe(() => {}).unsubscribe();


        let lastUpdateData = new Array<ProjectLastUpdates>();

        // Proj to keep
        let prj1 = new ProjectLastUpdates();
        prj1.name = 'key1';
        prj1.last_modified = 456;
        prj1.username = 'foo';
        prj1.applications = new Array<LastModification>();
        prj1.pipelines = new Array<LastModification>();

        // App not modified
        let app1 = new LastModification();
        app1.last_modified = 123;
        app1.name = 'app1';

        // Modified to keep
        let app2 = new LastModification();
        app2.last_modified = 456;
        app2.username = 'fooApp';
        app2.name = 'app2';

        prj1.applications.push(app1);
        prj1.applications.push(app2);

        // Pip not updated
        let pip1 = new LastModification();
        pip1.last_modified = 123;
        pip1.name = 'pip1';

        // Pip to keep
        let pip2 = new LastModification();
        pip2.last_modified = 456;
        pip2.name = 'pip2';
        pip2.username = 'fooApp';

        prj1.pipelines.push(pip1);
        prj1.pipelines.push(pip2);

        // Proj to delete
        let prj2 = new ProjectLastUpdates();
        prj2.name = 'key2';
        prj2.last_modified = 456;
        prj2.applications = new Array<LastModification>();
        prj2.pipelines = new Array<LastModification>();

        // App to delete
        let app3 = new LastModification();
        app3.name = 'app3';
        app3.last_modified = 456;
        app3.username = 'bar';

        prj2.applications.push(app3);

        let pip3 = new LastModification();
        pip3.name = 'pip3';
        app3.last_modified = 456;
        pip3.username = 'bar';

        prj2.pipelines.push(pip3);

        lastUpdateData.push(prj1, prj2);

        let AuthStore = injector.get(AuthentificationStore);
        let user = new User();
        AuthStore.addUser(user, true);

        let appService = injector.get(AppService);

        appService.updateCache(lastUpdateData);

        projectStore.getProjects().subscribe(projs => {

        }).unsubscribe();

         */

    }));

});
class MockActivatedRoutes extends ActivatedRoute {

    constructor() {
        super();
        this.snapshot = new ActivatedRouteSnapshot();
        this.snapshot.params = {
            'key': 'key1',
            'appName': 'app2',
            'pipName': 'pip2'
        };
    }
}
