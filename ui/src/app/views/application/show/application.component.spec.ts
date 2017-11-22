/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {Response, ResponseOptions, ResponseType} from '@angular/http';
import {Router, ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';
import {ApplicationShowComponent} from './application.component';
import {ApplicationStore} from '../../../service/application/application.store';
import {ApplicationService} from '../../../service/application/application.service';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../../../shared/shared.module';
import {ServicesModule} from '../../../service/services.module';
import {Observable} from 'rxjs/Observable';
import {Injector} from '@angular/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {ProjectStore} from '../../../service/project/project.store';
import {ProjectService} from '../../../service/project/project.service';
import {EnvironmentService} from '../../../service/environment/environment.service';
import {PipelineService} from '../../../service/pipeline/pipeline.service';
import {VariableService} from '../../../service/variable/variable.service';
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
import {Notification} from '../../../model/notification.model';
import {NotificationEvent} from './notifications/notification.event';
import {Pipeline} from '../../../model/pipeline.model';
import {Environment} from '../../../model/environment.model';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';

describe('CDS: Application', () => {

    let injector: Injector;
    let appStore: ApplicationStore;
    let router: Router;
    let prjStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                AuthentificationStore,
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                PipelineService,
                VariableService,
                EnvironmentService,
                ApplicationWorkflowService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: Router, useClass: MockRouter},
                { provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser,
            ],
            imports : [
                ApplicationModule,
                ServicesModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
        appStore = injector.get(ApplicationStore);
        router = injector.get(Router);
        prjStore = injector.get(ProjectStore);
    });

    afterEach(() => {
        injector = undefined;
        appStore = undefined;
        router = undefined;
        prjStore = undefined;
    });

    it('Load component + load application', fakeAsync( () => {

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
        const http = TestBed.get(HttpTestingController);

        spyOn(appStore, 'updateRecentApplication');
        spyOn(router, 'navigate');

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/key1/application/app1';
        })).flush({'name': 'app1'}, { status: 404, statusText: 'App does not exist'});

        expect(appStore.updateRecentApplication).not.toHaveBeenCalled();
        expect(router.navigate).toHaveBeenCalledWith(['/project', 'key1']);
    }));

    it('should run add variable', fakeAsync( () => {
        let call = 0;

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

    it('should run add/update/delete notification', fakeAsync( () => {
        let call = 0;

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

        spyOn(appStore, 'addNotifications').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        let n: Notification = new Notification();
        n.pipeline = new Pipeline();
        n.pipeline.name = 'pip1';
        n.environment = new Environment();
        n.environment.name = 'production';
        let notifs = new Array<Notification>();
        notifs.push(n);
        fixture.componentInstance.notificationEvent(new NotificationEvent('add', notifs));
        expect(appStore.addNotifications).toHaveBeenCalledWith('key1', 'app1', notifs);

        spyOn(appStore, 'updateNotification').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        fixture.componentInstance.notificationEvent(new NotificationEvent('update', notifs));
        expect(appStore.updateNotification).toHaveBeenCalledWith('key1', 'app1', 'pip1', n);

        spyOn(appStore, 'deleteNotification').and.callFake(() => {
            let app: Application = new Application();
            return Observable.of(app);
        });

        fixture.componentInstance.notificationEvent(new NotificationEvent('delete', notifs));
        expect(appStore.deleteNotification).toHaveBeenCalledWith('key1', 'app1', 'pip1', 'production');
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
    error(title: string, msg: string) {

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
        this.queryParams = Observable.of({key: 'key1', appName: 'app1', version: 0, branch: 'master'});

        this.snapshot = new ActivatedRouteSnapshot();

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project: project
        };
        this.snapshot.queryParams = {key: 'key1', appName: 'app1', version: 0, branch: 'master'};

        this.data = Observable.of({ project: project });
    }
}
