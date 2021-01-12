import { HttpClientTestingModule } from '@angular/common/http/testing';
import { Injector } from '@angular/core';
import { fakeAsync, getTestBed, TestBed } from '@angular/core/testing';
import { ActivatedRoute, ActivatedRouteSnapshot, Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ApplicationService } from 'app/service/application/application.service';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { MonitoringService } from 'app/service/monitoring/monitoring.service';
import { RouterService } from 'app/service/router/router.service';
import { UserService } from 'app/service/user/user.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AddPipeline } from 'app/store/pipelines.action';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Application } from '../../../model/application.model';
import { Pipeline } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { EnvironmentService } from '../../../service/environment/environment.service';
import { NavbarService } from '../../../service/navbar/navbar.service';
import { PipelineService } from '../../../service/pipeline/pipeline.service';
import { ProjectService } from '../../../service/project/project.service';
import { ProjectStore } from '../../../service/project/project.store';
import { VariableService } from '../../../service/variable/variable.service';
import { SharedModule } from '../../../shared/shared.module';
import { ToastService } from '../../../shared/toast/ToastService';
import { PipelineModule } from '../pipeline.module';
import { PipelineAddComponent } from './pipeline.add.component';

describe('CDS: Pipeline Add Component', () => {

    let injector: Injector;
    let store: Store;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ApplicationService,
                EnvironmentService,
                ProjectStore,
                ProjectService,
                MonitoringService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                { provide: Router, useClass: MockRouter },
                { provide: ToastService, useClass: MockToast },
                TranslateService,
                TranslateLoader,
                TranslateParser,
                NavbarService,
                PipelineService,
                EnvironmentService,
                VariableService,
                WorkflowService,
                WorkflowRunService,
                UserService,
                RouterService,
                AuthenticationService
            ],
            imports: [
                PipelineModule,
                NgxsStoreModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        }).compileComponents();

        injector = getTestBed();
        store = injector.get(Store);
    });

    afterEach(() => {
        injector = undefined;
        store = undefined;
    });

    it('should create an empty pipeline', fakeAsync(() => {

        // Create component
        let fixture = TestBed.createComponent(PipelineAddComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project: Project = new Project();
        project.key = 'key1';
        project.applications = new Array<Application>();
        let app1 = new Application();
        app1.name = 'app1';
        let app2 = new Application();
        app2.name = 'app2';
        project.applications.push(app1, app2);

        fixture.componentInstance.project = project;
        fixture.componentInstance.newPipeline = new Pipeline();
        fixture.componentInstance.newPipeline.name = 'myPip';

        spyOn(store, 'dispatch').and.callFake(() => of(null));

        fixture.componentInstance.createPipeline();
        expect(store.dispatch).toHaveBeenCalledWith(new AddPipeline({
            projectKey: 'key1',
            pipeline: fixture.componentInstance.newPipeline
        }));
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
        this.params = of({ key: 'key1', appName: 'app1' });
        this.queryParams = of({ key: 'key1', appName: 'app1' });

        this.snapshot = new ActivatedRouteSnapshot();

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project
        };

        this.data = of({ project });
    }
}
