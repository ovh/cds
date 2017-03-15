/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, tick, getTestBed} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {ApplicationModule} from '../../../../application.module';
import {SharedModule} from '../../../../../../shared/shared.module';
import {RouterTestingModule} from '@angular/router/testing';
import {ApplicationWorkflowItemComponent} from './application.workflow.item.component';
import {WorkflowItem} from '../../../../../../model/application.workflow.model';
import {Trigger} from '../../../../../../model/trigger.model';
import {Environment} from '../../../../../../model/environment.model';
import {Parameter} from '../../../../../../model/parameter.model';
import {Injector} from '@angular/core';
import {ApplicationPipelineService} from '../../../../../../service/application/pipeline/application.pipeline.service';
import {Router, NavigationExtras} from '@angular/router';
import {Observable} from 'rxjs/Rx';
import {PipelineBuild, Pipeline, PipelineRunRequest} from '../../../../../../model/pipeline.model';
import {Project} from '../../../../../../model/project.model';
import {Application} from '../../../../../../model/application.model';
import {TranslateParser, TranslateService, TranslateLoader} from 'ng2-translate';
import {PipelineStore} from '../../../../../../service/pipeline/pipeline.store';
import {PipelineService} from '../../../../../../service/pipeline/pipeline.service';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Map} from 'immutable';
import {ApplicationStore} from '../../../../../../service/application/application.store';
import {ApplicationService} from '../../../../../../service/application/application.service';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ProjectService} from '../../../../../../service/project/project.service';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {Scheduler} from '../../../../../../model/scheduler.model';

describe('CDS: Application Workflow Item', () => {

    let injector: Injector;
    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: APP_BASE_HREF, useValue: '/'},
                ApplicationPipelineService,
                {provide: Router, useClass: MockRouter},
                TranslateParser, TranslateService, TranslateLoader,
                PipelineStore, PipelineService,
                ApplicationStore, ApplicationService,
                ProjectStore, ProjectService,
                {provide: ToastService, useClass: MockToast}
            ],
            imports: [
                ApplicationModule,
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });

    it('should run a pipeline with parent information', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let appFilter = {branch: 'master'};
        fixture.componentInstance.applicationFilter = appFilter;

        let workflowItem = new WorkflowItem();
        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';

        // env
        workflowItem.environment = new Environment();
        workflowItem.environment.name = 'prod';
        workflowItem.environment.id = 4;

        // parent
        workflowItem.parent = {application_id: 1, pipeline_id: 2, environment_id: 3, branch: 'master', buildNumber: 123, version: 123};

        // trigger
        workflowItem.trigger = new Trigger();
        workflowItem.trigger.manual = false;
        workflowItem.trigger.parameters = new Array<Parameter>();
        workflowItem.trigger.src_application = workflowItem.application;
        workflowItem.trigger.src_pipeline = workflowItem.pipeline;
        workflowItem.trigger.src_environment = workflowItem.environment;
        let param = new Parameter();
        param.name = 'foo';
        param.value = 'barr';
        param.type = 'string';
        workflowItem.trigger.parameters.push(param);

        fixture.componentInstance.project = workflowItem.project;
        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;

        fixture.detectChanges();
        tick(250);

        let appPipStore = injector.get(ApplicationPipelineService);
        spyOn(appPipStore, 'run').and.callFake(() => {
            let pb = new PipelineBuild();
            pb.build_number = 12;
            pb.application = workflowItem.application;
            pb.pipeline = workflowItem.pipeline;
            pb.environment = workflowItem.environment;
            return Observable.of(pb);
        });
        fixture.componentInstance.runPipeline();

        let request: PipelineRunRequest = new PipelineRunRequest();
        request.env = workflowItem.environment;
        request.parameters = workflowItem.trigger.parameters;
        request.parent_application_id = 1;
        request.parent_build_number = 123;
        request.parent_environment_id = 3;
        request.parent_pipeline_id = 2;

        expect(appPipStore.run).toHaveBeenCalledWith('key1', 'app1', 'pip1', request);

    }));

    it('should run a pipeline without parent', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let appFilter = {branch: 'master'};
        fixture.componentInstance.applicationFilter = appFilter;

        let workflowItem = new WorkflowItem();
        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';
        workflowItem.pipeline.parameters = new Array<Parameter>();
        let paramPip = new Parameter();
        paramPip.name = 'fooPip';
        paramPip.value = 'barrPip';
        paramPip.type = 'string';
        workflowItem.pipeline.parameters.push(paramPip);

        // env
        workflowItem.environment = new Environment();
        workflowItem.environment.name = 'prod';
        workflowItem.environment.id = 4;

        // trigger
        workflowItem.trigger = new Trigger();
        workflowItem.trigger.manual = false;
        workflowItem.trigger.parameters = new Array<Parameter>();
        let param = new Parameter();
        param.name = 'foo';
        param.value = 'barr';
        param.type = 'string';
        workflowItem.trigger.parameters.push(param);
        workflowItem.trigger.src_application = workflowItem.application;
        workflowItem.trigger.src_pipeline = workflowItem.pipeline;
        workflowItem.trigger.src_environment = workflowItem.environment;

        fixture.componentInstance.project = workflowItem.project;
        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;

        fixture.detectChanges();
        tick(250);

        let appPipStore = injector.get(ApplicationPipelineService);
        spyOn(appPipStore, 'run').and.callFake(() => {
            let pb = new PipelineBuild();
            pb.build_number = 12;
            pb.application = workflowItem.application;
            pb.pipeline = workflowItem.pipeline;
            pb.environment = workflowItem.environment;
            return Observable.of(pb);
        });
        fixture.componentInstance.runPipeline();

        let request: PipelineRunRequest = new PipelineRunRequest();
        request.env = workflowItem.environment;
        request.parameters = workflowItem.pipeline.parameters;

        expect(appPipStore.run).toHaveBeenCalledWith('key1', 'app1', 'pip1', request);
    }));

    it('should not run a manual triggered pipeline', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let appFilter = {branch: 'master'};
        fixture.componentInstance.applicationFilter = appFilter;

        let workflowItem = new WorkflowItem();
        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';
        workflowItem.pipeline.parameters = new Array<Parameter>();
        let paramPip = new Parameter();
        paramPip.name = 'fooPip';
        paramPip.value = 'barrPip';
        paramPip.type = 'string';
        workflowItem.pipeline.parameters.push(paramPip);

        // env
        workflowItem.environment = new Environment();
        workflowItem.environment.name = 'prod';
        workflowItem.environment.id = 4;

        // trigger
        workflowItem.trigger = new Trigger();
        workflowItem.trigger.manual = true;
        workflowItem.trigger.src_application = workflowItem.application;
        workflowItem.trigger.src_pipeline = workflowItem.pipeline;
        workflowItem.trigger.src_environment = workflowItem.environment;

        fixture.componentInstance.project = workflowItem.project;
        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;

        fixture.detectChanges();
        tick(250);

        spyOn(fixture.componentInstance, 'runWithParameters').and.callFake(() => {
            return true;
        });

        fixture.componentInstance.runPipeline();

        expect(fixture.componentInstance.runWithParameters).toHaveBeenCalled();
    }));

    it('should not run a non triggered pipeline with empty parameter', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let appFilter = {branch: 'master'};
        fixture.componentInstance.applicationFilter = appFilter;

        let workflowItem = new WorkflowItem();
        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';
        workflowItem.pipeline.parameters = new Array<Parameter>();
        let paramPip = new Parameter();
        paramPip.name = 'fooPip';
        paramPip.value = '';
        paramPip.type = 'string';
        workflowItem.pipeline.parameters.push(paramPip);

        // env
        workflowItem.environment = new Environment();
        workflowItem.environment.name = 'prod';
        workflowItem.environment.id = 4;

        // trigger
        workflowItem.trigger = new Trigger();
        workflowItem.trigger.manual = false;
        workflowItem.trigger.src_application = workflowItem.application;
        workflowItem.trigger.src_pipeline = workflowItem.pipeline;
        workflowItem.trigger.src_environment = workflowItem.environment;

        fixture.componentInstance.project = workflowItem.project;
        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;

        fixture.detectChanges();
        tick(250);

        spyOn(fixture.componentInstance, 'runWithParameters').and.callFake(() => {
            return true;
        });
        fixture.componentInstance.runPipeline();

        expect(fixture.componentInstance.runWithParameters).toHaveBeenCalled();
    }));

    it('should load manual run data without parent information', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let appFilter = {branch: 'master'};
        fixture.componentInstance.applicationFilter = appFilter;

        let workflowItem = new WorkflowItem();
        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';
        workflowItem.pipeline.parameters = new Array<Parameter>();
        let paramPip = new Parameter();
        paramPip.name = 'fooPip';
        paramPip.value = '';
        paramPip.type = 'string';
        workflowItem.pipeline.parameters.push(paramPip);

        // env
        workflowItem.environment = new Environment();
        workflowItem.environment.name = 'prod';
        workflowItem.environment.id = 4;


        fixture.componentInstance.project = workflowItem.project;
        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;
        fixture.componentInstance.launchModal = new SemanticModalComponent();

        fixture.detectChanges();
        tick(250);




        let pipStore: PipelineStore = injector.get(PipelineStore);
        spyOn(pipStore, 'getPipelines').and.callFake(() => {
            let mapApp: Map<string, Pipeline> = Map<string, Pipeline>();
            let pip: Pipeline = new Pipeline();
            pip.name = 'pip1';
            pip.parameters = new Array<Parameter>();
            let p1 = new Parameter();
            p1.name = 'foo';
            pip.parameters.push(p1);
            return Observable.of(mapApp.set('key1-pip1', pip));
        });

        spyOn(fixture.componentInstance.launchModal, 'show').and.callFake(() => true);

        fixture.componentInstance.runWithParameters();

        // Check Git params
        expect(fixture.componentInstance.launchGitParams.length).toBe(1);
        expect(fixture.componentInstance.launchGitParams[0].name).toBe('git.branch');
        expect(fixture.componentInstance.launchGitParams[0].value).toBe(appFilter.branch);

        // Check parent
        expect(fixture.componentInstance.launchParentBuildNumber).toBeFalsy();

        // Check run parameter
        expect(fixture.componentInstance.launchPipelineParams.length).toBe(1);
        expect(fixture.componentInstance.launchPipelineParams[0].name).toBe('foo');

        tick(150);
        // Open modal?
        expect(fixture.componentInstance.launchModal.show).toHaveBeenCalled();
    }));

    it('should load manual run data with parent information', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let appFilter = {branch: 'master'};
        fixture.componentInstance.applicationFilter = appFilter;

        let workflowItem = new WorkflowItem();
        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';
        workflowItem.pipeline.parameters = new Array<Parameter>();
        let paramPip = new Parameter();
        paramPip.name = 'fooPip';
        paramPip.value = '';
        paramPip.type = 'string';
        workflowItem.pipeline.parameters.push(paramPip);

        workflowItem.parent = {version: 1, application_id: 1, branch: 'master', buildNumber: 1, pipeline_id: 1, environment_id: 1};

        // env
        workflowItem.environment = new Environment();
        workflowItem.environment.name = 'prod';
        workflowItem.environment.id = 4;

        // trigger
        workflowItem.trigger = new Trigger();
        workflowItem.trigger.id = 1;
        workflowItem.trigger.parameters = new Array<Parameter>();
        workflowItem.trigger.parameters.push(createParam('onlyInTrigger'));
        workflowItem.trigger.parameters.push(createParam('commonParam'));
        workflowItem.trigger.src_application = workflowItem.application;
        workflowItem.trigger.src_pipeline = workflowItem.pipeline;
        workflowItem.trigger.src_environment = workflowItem.environment;


        fixture.componentInstance.project = workflowItem.project;
        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;
        fixture.componentInstance.launchModal = new SemanticModalComponent();

        fixture.detectChanges();
        tick(250);




        let pipStore: PipelineStore = injector.get(PipelineStore);
        spyOn(pipStore, 'getPipelines').and.callFake(() => {
            let mapApp: Map<string, Pipeline> = Map<string, Pipeline>();
            let pip: Pipeline = new Pipeline();
            pip.name = 'pip1';
            pip.parameters = new Array<Parameter>();
            pip.parameters.push(createParam('commonParam', 'customValue'));
            pip.parameters.push(createParam('onlyInPip'));
            return Observable.of(mapApp.set('key1-pip1', pip));
        });

        let appPipService: ApplicationPipelineService = injector.get(ApplicationPipelineService);
        spyOn(appPipService, 'buildHistory').and.callFake(() => {
           let pbs = new Array<PipelineBuild>();
           pbs.push(createPipelineBuild(1));
            pbs.push(createPipelineBuild(2));
            pbs.push(createPipelineBuild(3));
            return Observable.of(pbs);
        });

        spyOn(fixture.componentInstance.launchModal, 'show').and.callFake(() => true);

        fixture.componentInstance.runWithParameters();

        // Check Git params
        expect(fixture.componentInstance.launchGitParams.length).toBe(1);
        expect(fixture.componentInstance.launchGitParams[0].name).toBe('git.branch');
        expect(fixture.componentInstance.launchGitParams[0].value).toBe(appFilter.branch);

        // Check parent
        expect(fixture.componentInstance.launchParentBuildNumber).toBe(1);

        // Check run parameter
        expect(fixture.componentInstance.launchPipelineParams.length).toBe(2);
        expect(fixture.componentInstance.launchPipelineParams[0].name).toBe('commonParam');
        expect(fixture.componentInstance.launchPipelineParams[0].value).toBe('commonParam-Value');
        expect(fixture.componentInstance.launchPipelineParams[1].name).toBe('onlyInPip');

        tick(150);
        // Open modal?
       expect(fixture.componentInstance.launchModal.show).toHaveBeenCalled();
    }));

    it('should add/update/delete trigger', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init modal
        fixture.componentInstance.createTriggerModal = new SemanticModalComponent();
        fixture.componentInstance.editTriggerModal = new SemanticModalComponent();

        // Init workflow item
        let workflowItem = new WorkflowItem();

        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';
        workflowItem.pipeline.type = 'build';

        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;
        fixture.componentInstance.project = workflowItem.project;

        fixture.detectChanges();
        tick(250);

        fixture.componentInstance.triggerInModal = new Trigger();
        fixture.componentInstance.triggerInModal.src_application = workflowItem.application;
        fixture.componentInstance.triggerInModal.src_pipeline = workflowItem.pipeline;

        // Add trigger

        let appStore: ApplicationStore = injector.get(ApplicationStore);
        spyOn(appStore, 'addTrigger').and.callFake(() => {
            return Observable.of(workflowItem.application);
        });
        spyOn(fixture.componentInstance.createTriggerModal, 'hide').and.callFake(() => true);
        fixture.componentInstance.triggerEvent('add');
        expect(appStore.addTrigger).toHaveBeenCalledWith('key1', 'app1', 'pip1', fixture.componentInstance.triggerInModal);
        expect(fixture.componentInstance.createTriggerModal.hide).toHaveBeenCalled();

        spyOn(fixture.componentInstance.editTriggerModal, 'hide').and.callFake(() => true);

        // Update trigger
        spyOn(appStore, 'updateTrigger').and.callFake(() => {
            return Observable.of(workflowItem.application);
        });
        fixture.componentInstance.triggerEvent('update');
        expect(appStore.updateTrigger).toHaveBeenCalledWith('key1', 'app1', 'pip1', fixture.componentInstance.triggerInModal);


        // Delete trigger
        spyOn(appStore, 'removeTrigger').and.callFake(() => {
            return Observable.of(workflowItem.application);
        });
        fixture.componentInstance.triggerEvent('delete');
        expect(appStore.removeTrigger).toHaveBeenCalledWith('key1', 'app1', 'pip1', fixture.componentInstance.triggerInModal);


        expect(fixture.componentInstance.editTriggerModal.hide).toHaveBeenCalledTimes(2);
    }));

    it('should add/update/delete a scheduler', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationWorkflowItemComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init modal
        fixture.componentInstance.createSchedulerModal = new SemanticModalComponent();

        // Init workflow item
        let workflowItem = new WorkflowItem();

        // project
        workflowItem.project = new Project();
        workflowItem.project.key = 'key1';

        // application
        workflowItem.application = new Application();
        workflowItem.application.name = 'app1';
        workflowItem.application.id = 6;

        // pipeline
        workflowItem.pipeline = new Pipeline();
        workflowItem.pipeline.name = 'pip1';
        workflowItem.pipeline.type = 'build';

        fixture.componentInstance.workflowItem = workflowItem;
        fixture.componentInstance.application = workflowItem.application;
        fixture.componentInstance.project = workflowItem.project;

        fixture.detectChanges();
        tick(250);

        fixture.componentInstance.newScheduler = new Scheduler();
        fixture.componentInstance.newScheduler.crontab = '* * * * *';

        // Add scheduler
        let appStore: ApplicationStore = injector.get(ApplicationStore);
        spyOn(appStore, 'addScheduler').and.callFake(() => {
            return Observable.of(workflowItem.application);
        });
        spyOn(fixture.componentInstance.createSchedulerModal, 'hide').and.callFake(() => true);

        fixture.componentInstance.schedulerEvent('add', fixture.componentInstance.newScheduler);
        expect(appStore.addScheduler).toHaveBeenCalledWith('key1', 'app1', 'pip1', fixture.componentInstance.newScheduler);
        expect(fixture.componentInstance.createSchedulerModal.hide).toHaveBeenCalled();

        // Update scheduler
        spyOn(appStore, 'updateScheduler').and.callFake(() => {
            return Observable.of(workflowItem.application);
        });
        fixture.componentInstance.schedulerEvent('update', fixture.componentInstance.newScheduler);
        expect(appStore.updateScheduler).toHaveBeenCalledWith('key1', 'app1', 'pip1', fixture.componentInstance.newScheduler);


        // Delete scheduler
        spyOn(appStore, 'deleteScheduler').and.callFake(() => {
            return Observable.of(workflowItem.application);
        });
        fixture.componentInstance.schedulerEvent('delete', fixture.componentInstance.newScheduler);
        expect(appStore.deleteScheduler).toHaveBeenCalledWith('key1', 'app1', 'pip1', fixture.componentInstance.newScheduler);
    }));
});

function createPipelineBuild(version: number): PipelineBuild {
    let pb = new PipelineBuild();
    pb.version = version;
    pb.build_number = version;
    return pb;
}
function createParam(name: string, value?: string): Parameter {
    let p = new Parameter();
    p.name = name;
    if (value) {
        p.value = value;
    } else {
        p.value = name + '-Value';
    }
    return p;
}

class MockToast {
    success(t: string, m: string) {

    }
}

class MockRouter {
    navigate(commands: any[], extras?: NavigationExtras): Promise<boolean> {
        return Promise.resolve(true);
    }
}
