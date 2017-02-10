/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {ActivatedRoute} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {Observable} from 'rxjs/Rx';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {PipelineService} from '../../../../../service/pipeline/pipeline.service';
import {PipelineStore} from '../../../../../service/pipeline/pipeline.store';
import {PipelineModule} from '../../../pipeline.module';
import {PipelineStageComponent} from './pipeline.stage.component';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {SharedModule} from '../../../../../shared/shared.module';
import {Stage} from '../../../../../model/stage.model';
import {Job} from '../../../../../model/job.model';
import {Action} from '../../../../../model/action.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {Parameter} from '../../../../../model/parameter.model';
import {PrerequisiteEvent} from '../../../../../shared/prerequisites/prerequisite.event.model';
import {Prerequisite} from '../../../../../model/prerequisite.model';
import {Project} from '../../../../../model/project.model';
import {ActionEvent} from '../../../../../shared/action/action.event.model';

describe('CDS: Stage', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: XHRBackend, useClass: MockBackend},
                PipelineService,
                PipelineStore,
                {provide: ActivatedRoute, useClass: MockActivatedRoutes},
                {provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports: [
                PipelineModule,
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(XHRBackend);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });

    it('should load component + change stage', fakeAsync(() => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "app1" }'})));
                    break;
            }

        });

        // Create component
        let fixture = TestBed.createComponent(PipelineStageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        /// Init pipeline
        let p = new Pipeline();
        fixture.componentInstance.pipeline = p;

        // Init stage
        let s = new Stage();
        s.id = 1;

        let jPreInit = new Job();
        jPreInit.pipeline_action_id = 1;

        let j = new Job();
        j.pipeline_action_id = 1;
        j.action = new Action();
        j.action.name = 'act1';
        s.jobs = new Array<Job>();
        s.jobs.push(j);

        fixture.componentInstance.selectedJob = jPreInit;
        fixture.componentInstance.stage = s;

        fixture.componentInstance.ngOnInit();

        expect(fixture.componentInstance.selectedJob.action.name).toBe('act1');
        expect(fixture.componentInstance.editableStage.prerequisites).toBeTruthy();


        let s2 = new Stage();
        s2.id = 2;

        let j2 = new Job();
        j2.pipeline_action_id = 2;
        s2.jobs = new Array<Job>();
        s2.jobs.push(j2);

        fixture.componentInstance.stage = s2;
        fixture.componentInstance.ngDoCheck();

        expect(fixture.componentInstance.selectedJob.pipeline_action_id).toBe(2);

    }));

    it('should load component + select 1st job + init available prerequisite', fakeAsync(() => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "app1" }'})));
                    break;
            }

        });

        // Create component
        let fixture = TestBed.createComponent(PipelineStageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init stage
        let s = new Stage();

        let j = new Job();
        j.pipeline_action_id = 1;
        j.action = new Action();
        j.action.name = 'act1';
        s.jobs = new Array<Job>();
        s.jobs.push(j);
        fixture.componentInstance.editableStage = s;

        // Init Pipeline
        let p = new Pipeline();
        p.parameters = new Array<Parameter>();

        let param = new Parameter();
        param.name = 'param1';
        p.parameters.push(param);
        fixture.componentInstance.pipeline = p;

        fixture.componentInstance.ngOnInit();

        expect(fixture.componentInstance.selectedJob.action.name).toBe('act1');
        expect(fixture.componentInstance.availablePrerequisites.length).toBe(2);
        expect(fixture.componentInstance.availablePrerequisites[0].parameter).toBe('git.branch');
        expect(fixture.componentInstance.availablePrerequisites[1].parameter).toBe('param1');
    }));

    it('should add and delete prerequisite', fakeAsync(() => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "app1" }'})));
                    break;
            }

        });

        // Create component
        let fixture = TestBed.createComponent(PipelineStageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init stage
        let s = new Stage();
        fixture.componentInstance.editableStage = s;

        let eventAdd = new PrerequisiteEvent('add', new Prerequisite());
        eventAdd.prerequisite.parameter = 'git.branch';
        eventAdd.prerequisite.expected_value = 'master';

        fixture.componentInstance.prerequisiteEvent(eventAdd);
        // add twice
        fixture.componentInstance.prerequisiteEvent(eventAdd);

        expect(fixture.componentInstance.editableStage.prerequisites.length).toBe(1, 'Must have 1 prerequisite');
        expect(fixture.componentInstance.editableStage.prerequisites[0].parameter).toBe('git.branch');
        expect(fixture.componentInstance.editableStage.prerequisites[0].expected_value).toBe('master');


        let eventDelete = new PrerequisiteEvent('delete', eventAdd.prerequisite);
        fixture.componentInstance.prerequisiteEvent(eventDelete);
        expect(fixture.componentInstance.editableStage.prerequisites.length).toBe(0, 'Must have 0 prerequisite');
    }));

    it('should add/update/delete a job', fakeAsync(() => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "pip", "stages": [] }'})));
        });

        // Create component
        let fixture = TestBed.createComponent(PipelineStageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init stage
        let s = new Stage();
        s.id = 123;
        fixture.componentInstance.editableStage = s;

        // Init project
        let proj = new Project();
        proj.key = 'key';
        fixture.componentInstance.project = proj;

        // Init pipeline
        let pip = new Pipeline();
        pip.name = 'pip';
        fixture.componentInstance.pipeline = pip;

        let jobToAdd = new Job();
        jobToAdd.action.name = 'New Job';
        jobToAdd.enabled = true;

        let pipStore = injector.get(PipelineStore);

        spyOn(pipStore, 'addJob').and.callFake(() => {
            return Observable.of(pip);
        });
        fixture.componentInstance.addJob();

        expect(pipStore.addJob).toHaveBeenCalledWith('key', 'pip', 123, jobToAdd);

        let event = new ActionEvent('update', jobToAdd.action);
        spyOn(pipStore, 'updateJob').and.callFake(() => {
            return Observable.of(pip);
        });
        fixture.componentInstance.selectedJob = jobToAdd;
        fixture.componentInstance.jobEvent(event);

        expect(pipStore.updateJob).toHaveBeenCalledWith('key', 'pip', 123, jobToAdd);


        event.type = 'delete';
        spyOn(pipStore, 'removeJob').and.callFake(() => {
            return Observable.of(pip);
        });
        fixture.componentInstance.jobEvent(event);

        expect(pipStore.removeJob).toHaveBeenCalledWith('key', 'pip', 123, jobToAdd);

    }));
    it('should update/delete a stage', fakeAsync( () => {

        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "pip", "stages": [] }'})));
        });

        // Create component
        let fixture = TestBed.createComponent(PipelineStageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init stage
        let s = new Stage();
        s.id = 123;
        fixture.componentInstance.editableStage = s;

        // Init project
        let proj = new Project();
        proj.key = 'key';
        fixture.componentInstance.project = proj;

        // Init pipeline
        let pip = new Pipeline();
        pip.name = 'pip';
        fixture.componentInstance.pipeline = pip;

        // UPDATE

        let pipStore = injector.get(PipelineStore);
        spyOn(pipStore, 'updateStage').and.callFake(() => {
            return Observable.of(pip);
        });
        fixture.componentInstance.stageEvent('update');
        expect(pipStore.updateStage).toHaveBeenCalledWith('key', 'pip', s);

        // DELETE

        spyOn(pipStore, 'removeStage').and.callFake(() => {
            return Observable.of(pip);
        });

        fixture.componentInstance.stageEvent('delete');
        expect(pipStore.removeStage).toHaveBeenCalledWith('key', 'pip', s);

    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({key: 'key1', appName: 'app1'});
        this.queryParams = Observable.of({key: 'key1', appName: 'app1'});
    }
}
