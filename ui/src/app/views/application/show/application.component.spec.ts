/* tslint:disable:no-unused-variable */

import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { Injector } from '@angular/core';
import { fakeAsync, getTestBed, TestBed, tick } from '@angular/core/testing';
import { ActivatedRoute, ActivatedRouteSnapshot, Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Variable } from 'app/model/variable.model';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import { AddApplicationVariable, DeleteApplicationVariable, UpdateApplicationVariable } from 'app/store/applications.action';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Application } from '../../../model/application.model';
import { Project } from '../../../model/project.model';
import { Usage } from '../../../model/usage.model';
import { ApplicationService } from '../../../service/application/application.service';
import { ApplicationStore } from '../../../service/application/application.store';
import { ApplicationWorkflowService } from '../../../service/application/application.workflow.service';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { EnvironmentService } from '../../../service/environment/environment.service';
import { NavbarService } from '../../../service/navbar/navbar.service';
import { PipelineService } from '../../../service/pipeline/pipeline.service';
import { ProjectService } from '../../../service/project/project.service';
import { ProjectStore } from '../../../service/project/project.store';
import { MonitoringService, ServicesModule, WorkflowRunService, WorkflowStore } from '../../../service/services.module';
import { VariableService } from '../../../service/variable/variable.service';
import { WorkflowService } from '../../../service/workflow/workflow.service';
import { SharedModule } from '../../../shared/shared.module';
import { ToastService } from '../../../shared/toast/ToastService';
import { ApplicationModule } from '../application.module';
import { ApplicationShowComponent } from './application.component';

describe('CDS: Application', () => {

    let injector: Injector;
    let appStore: ApplicationStore;
    let store: Store;
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
                Store
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
        });

        injector = getTestBed();
        appStore = injector.get(ApplicationStore);
        store = injector.get(Store);
        router = injector.get(Router);
        prjStore = injector.get(ProjectStore);
    });

    afterEach(() => {
        injector = undefined;
        appStore = undefined;
        store = undefined;
        router = undefined;
        prjStore = undefined;
    });

    it('Load component + load application', fakeAsync(() => {

        spyOn(appStore, 'updateRecentApplication');

        spyOn(store, 'select').and.callFake(() => {
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            return of(app);
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

        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/key1/application/app1';
        })).flush({ 'name': 'app1' }, { status: 404, statusText: 'App does not exist' });

        tick(250);

        expect(appStore.updateRecentApplication).not.toHaveBeenCalled();
        expect(router.navigate).toHaveBeenCalledWith(['/project', 'key1'], { queryParams: { tab: 'applications' } });
    }));

    it('should run add variable', fakeAsync(() => {
        let call = 0;

        spyOn(store, 'select').and.callFake(() => {
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            return of(app);
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(store, 'dispatch').and.callFake(() => {
            return of();
        });

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
        spyOn(store, 'select').and.callFake(() => {
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            return of(app);
        });

        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        spyOn(store, 'dispatch').and.callFake(() => {
            return of();
        });

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
        spyOn(store, 'select').and.callFake(() => {
            let app: Application = new Application();
            app.name = 'app1';
            app.usage = new Usage();
            return of(app);
        });
        // Create component
        let fixture = TestBed.createComponent(ApplicationShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();
        spyOn(store, 'dispatch').and.callFake(() => {
            return of();
        });

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
            project: project
        };
        this.snapshot.queryParams = { key: 'key1', appName: 'app1', version: 0, branch: 'master' };

        this.data = of({ project: project });
    }
}
