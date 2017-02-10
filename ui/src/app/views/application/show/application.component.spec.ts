/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Router, ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';
import {ApplicationShowComponent} from './application.component';
import {ApplicationStore} from '../../../service/application/application.store';
import {ApplicationService} from '../../../service/application/application.service';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../../../shared/shared.module';
import {Observable} from 'rxjs/Rx';
import {Injector} from '@angular/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {ProjectStore} from '../../../service/project/project.store';
import {ProjectService} from '../../../service/project/project.service';
import {ApplicationModule} from '../application.module';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {VariableEvent} from '../../../shared/variable/variable.event.model';
import {Variable} from '../../../model/variable.model';
import {Application} from '../../../model/application.model';
import {GroupPermission} from '../../../model/group.model';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {Map} from 'immutable';
import {Project} from '../../../model/project.model';
import {ApplicationWorkflowService} from '../../../service/application/application.workflow.service';

describe('CDS: Application', () => {

    let injector: Injector;
    let appStore: ApplicationStore;
    let backend: MockBackend;
    let router: Router;
    let prjStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                AuthentificationStore,
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                ApplicationWorkflowService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: Router, useClass: MockRouter},
                { provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                ApplicationModule,
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(XHRBackend);
        appStore = injector.get(ApplicationStore);
        router = injector.get(Router);
        prjStore = injector.get(ProjectStore);
    });

    afterEach(() => {
        injector = undefined;
        appStore = undefined;
        backend = undefined;
        router = undefined;
        prjStore = undefined;
    });

    it('Load component + load application', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '[]'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
                    break;
            }

        });

        spyOn(appStore, 'updateRecentApplication');

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            return Observable.of(mapApp.set('key1-app1', app));
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
        expect(fixture.componentInstance.project.key).toBe('key1');
        expect(appStore.updateRecentApplication).toHaveBeenCalled();
    }));

    it('Load component + load application with error', fakeAsync( () => {
        // Mock Http
        backend.connections.subscribe(connection => {
            connection.mockError(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
        });

        spyOn(appStore, 'updateRecentApplication');
        spyOn(router, 'navigate');


        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        expect(appStore.updateRecentApplication).not.toHaveBeenCalled();
        expect(router.navigate).toHaveBeenCalledWith(['/project', 'key1']);
    }));

    it('should run add variable', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '[]'})));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
                    break;
            }

        });

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            return Observable.of(mapApp.set('key1-app1', app));
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(appStore, 'addVariable').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        let v: Variable = new Variable();
        v.name = 'foo';
        fixture.componentInstance.variableEvent(new VariableEvent('add', v));
        expect(appStore.addVariable).toHaveBeenCalledWith('key1', 'app1', v);
    }));

    it('should run update variable', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '[]'})));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
                    break;
            }

        });

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            return Observable.of(mapApp.set('key1-app1', app));
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(appStore, 'updateVariable').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        let v: Variable = new Variable();
        v.name = 'foo';
        fixture.componentInstance.variableEvent(new VariableEvent('update', v));
        expect(appStore.updateVariable).toHaveBeenCalledWith('key1', 'app1', v);
    }));

    it('should run remove variable', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '[]'})));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
                    break;
            }

        });

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            return Observable.of(mapApp.set('key1-app1', app));
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(appStore, 'removeVariable').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        let v: Variable = new Variable();
        v.name = 'foo';
        fixture.componentInstance.variableEvent(new VariableEvent('delete', v));
        expect(appStore.removeVariable).toHaveBeenCalledWith('key1', 'app1', v);
    }));

    it('should run add permission', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '[]'})));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
                    break;
            }

        });

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            return Observable.of(mapApp.set('key1-app1', app));
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(appStore, 'addPermission').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 7;
        fixture.componentInstance.groupEvent(new PermissionEvent('add', gp));
        expect(appStore.addPermission).toHaveBeenCalledWith('key1', 'app1', gp);
    }));

    it('should run update permission', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '[]'})));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
                    break;
            }

        });

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            return Observable.of(mapApp.set('key1-app1', app));
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(appStore, 'updatePermission').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 7;
        fixture.componentInstance.groupEvent(new PermissionEvent('update', gp));
        expect(appStore.updatePermission).toHaveBeenCalledWith('key1', 'app1', gp);
    }));

    it('should run remove permission', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '[]'})));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "app1" }'})));
                    break;
            }

        });

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            return Observable.of(mapApp.set('key1-app1', app));
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(appStore, 'removePermission').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 7;
        fixture.componentInstance.groupEvent(new PermissionEvent('delete', gp));
        expect(appStore.removePermission).toHaveBeenCalledWith('key1', 'app1', gp);
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockRouter {
    public navigate() {
    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({key: 'key1', appName: 'app1'});
        this.queryParams = Observable.of({key: 'key1', appName: 'app1'});

        this.snapshot = new ActivatedRouteSnapshot();

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project: project
        };

        this.data = Observable.of({ project: project });
    }
}
