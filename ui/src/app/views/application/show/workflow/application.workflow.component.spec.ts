/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {RouterTestingModule} from '@angular/router/testing';
import {ApplicationWorkflowComponent} from './application.workflow.component';
import {ApplicationModule} from '../../application.module';
import {SharedModule} from '../../../../shared/shared.module';
import {ApplicationWorkflowService} from '../../../../service/application/application.workflow.service';
import {ProjectService} from '../../../../service/project/project.service';
import {PipelineService} from '../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../service/environment/environment.service';
import {VariableService} from '../../../../service/variable/variable.service';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {Project} from '../../../../model/project.model';
import {Application, ApplicationFilter} from '../../../../model/application.model';
import {XHRBackend} from '@angular/http';
import {MockBackend} from '@angular/http/testing';
import {Observable} from 'rxjs/Rx';
import {WorkflowItem, WorkflowStatusResponse} from '../../../../model/application.workflow.model';
import {PipelineBuild, Pipeline} from '../../../../model/pipeline.model';
import {Environment} from '../../../../model/environment.model';
import {Scheduler, SchedulerExecution} from '../../../../model/scheduler.model';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Application Workflow', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                MockBackend,
                { provide: APP_BASE_HREF, useValue: '/' },
                { provide: XHRBackend, useClass: MockBackend },
                ApplicationWorkflowService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService
            ],
            imports : [
                ApplicationModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(MockBackend);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });

    it('should load component', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init Input Data
        let p: Project = new Project();
        p.key = 'key1';
        p.name = 'projectName';

        let a: Application = new Application();
        a.repository_fullname = 'repoFullName';
        a.name = 'appName';

        let appFilter: ApplicationFilter = { branch: '', version: '1', remote: 'barremote' };

        fixture.componentInstance.project = p;
        fixture.componentInstance.application = a;
        fixture.componentInstance.applicationFilter = appFilter;

        // Create spy
        let workflowService: ApplicationWorkflowService = injector.get(ApplicationWorkflowService);
        spyOn(workflowService, 'getRemotes').and.callFake(() => {
            return Observable.of([{ 'name' : 'barremote', url: 'https://github.com/barremote/barremote.git' }]);
        });
        spyOn(workflowService, 'getBranches').and.callFake(() => {
            return Observable.of([{ 'display_id' : 'branche1', default: true}, { 'display_id' : 'branche2'}, { 'display_id' : 'master'},
                { 'display_id' : 'branche3' }]);
        });
        spyOn(workflowService, 'getVersions').and.callFake(() => {
            return Observable.of([1, 2, 3]);
        });

        // Run component initialisation
        fixture.componentInstance.ngOnInit();

        // Check
        expect(fixture.componentInstance.applicationFilter.branch).toBe('branche1');
        expect(JSON.stringify(fixture.componentInstance.versions)).toBe(JSON.stringify([' ', '1', '2', '3']));
        expect(JSON.stringify(fixture.componentInstance.remotes)).toBe(JSON.stringify([{
            'name' : 'barremote',
            url: 'https://github.com/barremote/barremote.git'
        }]));
        expect(JSON.stringify(fixture.componentInstance.branches)).toBe(JSON.stringify([
            {'display_id': 'branche1', 'default': true},
            {'display_id': 'branche2'},
            {'display_id': 'master'},
            {'display_id': 'branche3'}
        ]));
    }));

    it('should refresh the workflow', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Create input data
        let p: Project = new Project();
        p.key = 'key1';
        p.name = 'projectName';

        let a: Application = new Application();
        a.name = 'appName';
        a.id = 1;

        let e1: Environment = new Environment();
        e1.id = 1;
        e1.name = 'NoEnv';
        let e2: Environment = new Environment();
        e2.id = 2;
        e2.name = 'prod';

        // Create pipeline
        let buildPip: Pipeline = new Pipeline();
        buildPip.id = 1;
        buildPip.name = 'buildPipeline';
        let deploydPip: Pipeline = new Pipeline();
        deploydPip.id = 2;
        deploydPip.name = 'deployPipeline';

        // Create scheduler
        let s = new Scheduler();
        s.id = 1;
        s.next_execution = new SchedulerExecution();
        s.next_execution.execution_planned_date = '123';

        // Current workflow
        let currentWorkflow: Array<WorkflowItem> = new Array<WorkflowItem>();

        let rootItem1: WorkflowItem = new WorkflowItem();
        rootItem1.application = a;
        rootItem1.pipeline = buildPip;
        rootItem1.environment = e1;
        rootItem1.schedulers = new Array<Scheduler>();
        rootItem1.schedulers.push(s);

        let child1: WorkflowItem = new WorkflowItem();
        child1.application = a;
        child1.pipeline = deploydPip;
        child1.environment = e2;

        rootItem1.subPipelines = new Array<WorkflowItem>();
        rootItem1.subPipelines.push(child1);

        currentWorkflow.push(rootItem1);

        a.workflows = currentWorkflow;


        // Updated Application to apply
        let upApp: WorkflowStatusResponse = new WorkflowStatusResponse();

        upApp.schedulers = new Array<Scheduler>();
        let sUp = new Scheduler();
        sUp.id = 1;
        sUp.next_execution = new SchedulerExecution();
        sUp.next_execution.execution_planned_date = '456';
        upApp.schedulers.push(sUp);

        let pbs: Array<PipelineBuild> = new Array<PipelineBuild>();

        let pbItem1: PipelineBuild = new PipelineBuild();
        pbItem1.application = a;
        pbItem1.pipeline = buildPip;
        pbItem1.environment = e1;
        pbItem1.version = 6;

        let pbItem2: PipelineBuild = new PipelineBuild();
        pbItem2.application = a;
        pbItem2.pipeline = deploydPip;
        pbItem2.environment = e2;
        pbItem2.version = 5;

        pbs.push(pbItem1, pbItem2);
        upApp.builds = pbs;

        // Init component with input datas
        fixture.componentInstance.project = p;
        fixture.componentInstance.application = a;

        // Run test
        fixture.componentInstance.refreshWorkflow(upApp);

        tick(100);

        expect(fixture.componentInstance.application.workflows[0].pipeline.last_pipeline_build.version).toBe(6);
        expect(fixture.componentInstance.application.workflows[0].subPipelines[0].pipeline.last_pipeline_build.version).toBe(5);
        expect(fixture.componentInstance.application.workflows[0].schedulers[0].next_execution.execution_planned_date).toBe('456');
    }));

    it('should add staging env in workflow', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Create input data
        let p: Project = new Project();
        p.key = 'key1';
        p.name = 'projectName';

        let a: Application = new Application();
        a.name = 'appName';
        a.id = 1;

        let e0: Environment = new Environment();
        e0.id = 0;
        e0.name = 'NoEnv';
        let e1: Environment = new Environment();
        e1.id = 1;
        e1.name = 'staging';
        let e2: Environment = new Environment();
        e2.id = 2;
        e2.name = 'prod';
        let e3: Environment = new Environment();
        e3.id = 3;
        e3.name = 'preprod';
        p.environments = new Array<Environment>();
        p.environments.push(e1, e2, e3);


        // Create pipeline
        let deploydPip: Pipeline = new Pipeline();
        deploydPip.id = 1;
        deploydPip.type = 'deployment';
        deploydPip.name = 'deployPipeline';

        // Create workflow
        let currentWorkflow: Array<WorkflowItem> = new Array<WorkflowItem>();

        let rootItem1: WorkflowItem = new WorkflowItem();
        rootItem1.application = a;
        rootItem1.pipeline = deploydPip;
        rootItem1.environment = e0;

        currentWorkflow.push(rootItem1);

        a.workflows = currentWorkflow;

        // Updated Application to apply
        let upApp: WorkflowStatusResponse = new WorkflowStatusResponse();

        let pbs: Array<PipelineBuild> = new Array<PipelineBuild>();

        let pbItem1: PipelineBuild = new PipelineBuild();
        pbItem1.application = a;
        pbItem1.pipeline = deploydPip;
        pbItem1.environment = e1;
        pbItem1.version = 6;

        let pbItem2: PipelineBuild = new PipelineBuild();
        pbItem2.application = a;
        pbItem2.pipeline = deploydPip;
        pbItem2.environment = e2;
        pbItem2.version = 5;


        pbs.push(pbItem1, pbItem2);
        upApp.builds = pbs;

        fixture.componentInstance.project = p;
        fixture.componentInstance.application = a;

        // Run test
        fixture.componentInstance.refreshWorkflow(upApp);

        tick(100);

        expect(fixture.componentInstance.application.workflows.length).toBe(3);
        expect(fixture.componentInstance.application.workflows[0].pipeline.last_pipeline_build.version).toBe(6);
        expect(fixture.componentInstance.application.workflows[1].pipeline.last_pipeline_build.version).toBe(5);
        expect(JSON.stringify(fixture.componentInstance.application.workflows[0].environment)).toBe(JSON.stringify(e1));
        expect(JSON.stringify(fixture.componentInstance.application.workflows[1].environment)).toBe(JSON.stringify(e2));
        expect(JSON.stringify(fixture.componentInstance.application.workflows[2].environment)).toBe(JSON.stringify(e3));
    }));
});
