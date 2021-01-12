import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { Injector } from '@angular/core';
import { fakeAsync, getTestBed, TestBed, tick } from '@angular/core/testing';
import { ActivatedRoute, ActivatedRouteSnapshot, Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Permission } from 'app/model/permission.model';
import { Variable } from 'app/model/variable.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { UserService } from 'app/service/user/user.service';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import { AddApplicationVariable, DeleteApplicationVariable, UpdateApplicationVariable } from 'app/store/applications.action';
import { ApplicationStateModel } from 'app/store/applications.state';
import { ProjectStateModel } from 'app/store/project.state';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Application } from 'app/model/application.model';
import { Project } from 'app/model/project.model';
import { Usage } from 'app/model/usage.model';
import { ApplicationService } from 'app/service/application/application.service';
import { ApplicationStore } from 'app/service/application/application.store';
import { ApplicationWorkflowService } from 'app/service/application/application.workflow.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import {
    MonitoringService,
    RouterService,
    ServicesModule,
    WorkflowRunService,
    WorkflowStore
} from 'app/service/services.module';
import { VariableService } from 'app/service/variable/variable.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { SharedModule } from 'app/shared/shared.module';
import { ToastService } from 'app/shared/toast/ToastService';
import { ApplicationModule } from '../application.module';
import { ApplicationShowComponent } from './application.component';

describe('CDS: Application', () => {

    let injector: Injector;
    let appStore: ApplicationStore;
    let store: Store;
    let router: Router;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                PipelineService,
                VariableService,
                EnvironmentService,
                MonitoringService,
                NavbarService,
                ApplicationWorkflowService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                { provide: Router, useClass: MockRouter },
                { provide: ToastService, useClass: MockToast },
                TranslateService,
                TranslateLoader,
                TranslateParser,
                WorkflowStore,
                WorkflowService,
                WorkflowRunService,
                Store,
                UserService,
                RouterService,
                AuthenticationService
            ],
            imports: [
                ApplicationModule,
                NgxsStoreModule,
                ServicesModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        }).compileComponents();

        injector = getTestBed();
        appStore = injector.get(ApplicationStore);
        store = injector.get(Store);
        router = injector.get(Router);
    });

    afterEach(() => {
        injector = undefined;
        appStore = undefined;
        store = undefined;
        router = undefined;
    });

    it('Load component + load application', fakeAsync(() => {

        spyOn(appStore, 'updateRecentApplication');

        let callOrder = 0;
        spyOn(store, 'select').and.callFake(() => {
            if (callOrder === 0) {
                callOrder++;
                let state = new ProjectStateModel();
                let p = new Project();
                p.permissions = new Permission();
                p.permissions.writable = true;
                state.project = p;
                return of(state);
            }
            let state = new ApplicationStateModel();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            state.application = app;
            state.editMode = false;
            return of(state);
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        expect(fixture.componentInstance.application.name).toBe('app1');
        expect(appStore.updateRecentApplication).toHaveBeenCalled();

    }));

    it('Load component + load application with error', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);

        spyOn(appStore, 'updateRecentApplication');
        spyOn(router, 'navigate');

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/key1/application/app1')).flush({ name: 'app1' }, { status: 404, statusText: 'App does not exist' });

        tick(250);

        expect(appStore.updateRecentApplication).not.toHaveBeenCalled();
        expect(router.navigate).toHaveBeenCalledWith(['/project', 'key1'], { queryParams: { tab: 'applications' } });
    }));

    it('should run add variable', fakeAsync(() => {
        let call = 0;

        let callOrder = 0;
        spyOn(store, 'select').and.callFake(() => {
            if (callOrder === 0) {
                callOrder++;
                let state = new ProjectStateModel();
                let p = new Project();
                p.permissions = new Permission();
                p.permissions.writable = true;
                state.project = p;
                return of(state);
            }
            let state = new ApplicationStateModel();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            state.application = app;
            return of(state);
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(store, 'dispatch').and.callFake(() => of());

        let v: Variable = new Variable();
        v.name = 'foo';
        let project = new Project();
        project.key = 'key1';
        fixture.componentInstance.project = project;
        fixture.componentInstance.variableEvent(new VariableEvent('add', v));
        tick(250);
        expect(store.dispatch).toHaveBeenCalledWith(new AddApplicationVariable({
            projectKey: 'key1',
            applicationName: 'app1',
            variable: v
        }));
    }));

    it('should run update variable', fakeAsync(() => {
        let callOrder = 0;
        spyOn(store, 'select').and.callFake(() => {
            if (callOrder === 0) {
                callOrder++;
                let state = new ProjectStateModel();
                let p = new Project();
                p.permissions = new Permission();
                p.permissions.writable = true;
                state.project = p;
                return of(state);
            }
            let state = new ApplicationStateModel();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            state.application = app;
            return of(state);
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(store, 'dispatch').and.callFake(() => of());

        let v: Variable = new Variable();
        v.name = 'foo';
        let project = new Project();
        project.key = 'key1';
        fixture.componentInstance.project = project;
        fixture.componentInstance.variableEvent(new VariableEvent('update', v));
        tick(250);
        expect(store.dispatch).toHaveBeenCalledWith(new UpdateApplicationVariable({
            projectKey: 'key1',
            applicationName: 'app1',
            variableName: 'foo',
            variable: v
        }));
    }));

    it('should run remove variable', fakeAsync(() => {
        let callOrder = 0;
        spyOn(store, 'select').and.callFake(() => {
            if (callOrder === 0) {
                callOrder++;
                let state = new ProjectStateModel();
                let p = new Project();
                p.permissions = new Permission();
                p.permissions.writable = true;
                state.project = p;
                return of(state);
            }
            let state = new ApplicationStateModel();
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            state.application = app;
            return of(state);
        });
        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();
        spyOn(store, 'dispatch').and.callFake(() => of());

        let v: Variable = new Variable();
        v.name = 'foo';
        let project = new Project();
        project.key = 'key1';
        fixture.componentInstance.project = project;
        fixture.componentInstance.variableEvent(new VariableEvent('delete', v));
        tick(250);
        expect(store.dispatch).toHaveBeenCalledWith(new DeleteApplicationVariable({
            projectKey: 'key1',
            applicationName: 'app1',
            variable: v
        }));
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
        this.params = of({ key: 'key1', appName: 'app1' });
        this.queryParams = of({ key: 'key1', appName: 'app1', version: 0, branch: 'master' });

        this.snapshot = new ActivatedRouteSnapshot();

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project
        };
        this.snapshot.queryParams = { key: 'key1', appName: 'app1', version: 0, branch: 'master' };

        this.data = of({ project });
    }
}
