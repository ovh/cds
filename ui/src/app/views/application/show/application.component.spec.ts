/* tslint:disable:no-unused-variable */

import {HttpRequest} from '@angular/common/http';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {Injector} from '@angular/core';
import {fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {ActivatedRoute, ActivatedRouteSnapshot, Router} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateLoader, TranslateModule, TranslateParser, TranslateService} from '@ngx-translate/core';
import {Map} from 'immutable';
import 'rxjs/add/observable/of';
import {Observable} from 'rxjs/Observable';
import {Application} from '../../../model/application.model';
import {Environment} from '../../../model/environment.model';
import {GroupPermission} from '../../../model/group.model';
import {Notification} from '../../../model/notification.model';
import {Pipeline} from '../../../model/pipeline.model';
import {Project} from '../../../model/project.model';
import {Usage} from '../../../model/usage.model';
import {Variable} from '../../../model/variable.model';
import {ApplicationService} from '../../../service/application/application.service';
import {ApplicationStore} from '../../../service/application/application.store';
import {ApplicationWorkflowService} from '../../../service/application/application.workflow.service';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {EnvironmentService} from '../../../service/environment/environment.service';
import {NavbarService} from '../../../service/navbar/navbar.service';
import {PipelineService} from '../../../service/pipeline/pipeline.service';
import {ProjectService} from '../../../service/project/project.service';
import {ProjectStore} from '../../../service/project/project.store';
import {ServicesModule, WorkflowStore} from '../../../service/services.module';
import {VariableService} from '../../../service/variable/variable.service';
import {WorkflowService} from '../../../service/workflow/workflow.service';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {SharedModule} from '../../../shared/shared.module';
import {ToastService} from '../../../shared/toast/ToastService';
import {VariableEvent} from '../../../shared/variable/variable.event.model';
import {ApplicationModule} from '../application.module';
import {ApplicationShowComponent} from './application.component';
import {NotificationEvent} from './notifications/notification.event';

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
                NavbarService,
                ApplicationWorkflowService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: Router, useClass: MockRouter},
                { provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser,
                WorkflowStore,
                WorkflowService
            ],
            imports : [
                ApplicationModule,
                ServicesModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
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
            app.usage = new Usage();
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

        tick(250);

        expect(appStore.updateRecentApplication).not.toHaveBeenCalled();
        expect(router.navigate).toHaveBeenCalledWith(['/project', 'key1'], { queryParams: { tab: 'applications'}});
    }));

    it('should run add variable', fakeAsync( () => {
        let call = 0;

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
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
        tick(250);
        expect(appStore.addVariable).toHaveBeenCalledWith('key1', 'app1', v);
    }));

    it('should run update variable', fakeAsync( () => {

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
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
        tick(250);
        expect(appStore.updateVariable).toHaveBeenCalledWith('key1', 'app1', v);
    }));

    it('should run remove variable', fakeAsync( () => {

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
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
        tick(250);
        expect(appStore.removeVariable).toHaveBeenCalledWith('key1', 'app1', v);
    }));

    it('should run add permission', fakeAsync( () => {

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
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
            app.usage = new Usage();
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
            app.usage = new Usage();
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
        tick(250);
        expect(appStore.removePermission).toHaveBeenCalledWith('key1', 'app1', gp);
    }));

    it('should run add/update/delete notification', fakeAsync( () => {
        let call = 0;

        prjStore.getProjects('key1').subscribe(() => {}).unsubscribe();

        spyOn(appStore, 'getApplications').and.callFake( () => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
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
        tick(250);
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
