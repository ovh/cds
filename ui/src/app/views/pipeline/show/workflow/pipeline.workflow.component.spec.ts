/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {ActivatedRoute} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {Observable} from 'rxjs/Rx';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {PipelineWorkflowComponent} from './pipeline.workflow.component';
import {PipelineService} from '../../../../service/pipeline/pipeline.service';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {ToastService} from '../../../../shared/toast/ToastService';
import {PipelineModule} from '../../pipeline.module';
import {SharedModule} from '../../../../shared/shared.module';
import {Pipeline} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {Stage} from '../../../../model/stage.model';

describe('CDS: Pipeline Workflow', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                PipelineService,
                PipelineStore,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports : [
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

    it('should load component + add stage', fakeAsync( () => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "pip" }'})));
                    break;
                case 2:
                    connection.mockRespond(
                        new Response(
                            new ResponseOptions({ body : '{ "name": "pip", "stages": [ { "name" : "Stage 1"} ]}'})));
                    break;
            }

        });

        // Create component
        let fixture = TestBed.createComponent(PipelineWorkflowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init data
        let p = new Project();
        p.key = 'key';
        fixture.componentInstance.project = p;

        let pip = new Pipeline();
        pip.last_modified = 0;
        pip.name = 'pip';
        fixture.componentInstance.pipeline = pip;

        fixture.componentInstance.ngOnInit();

        let s = new Stage();
        s.enabled = true;
        s.name = 'Stage 1';

        let pipStore: PipelineStore = injector.get(PipelineStore);
        pipStore.createPipeline('key', pip).subscribe(() => {});


        let pipUpdated = new Pipeline();
        pip.last_modified = 123;
        pipUpdated.stages = new Array<Stage>();
        pipUpdated.stages.push(s);

        spyOn(pipStore, 'addStage').and.callFake(() => {


          return Observable.of(pipUpdated);
        });

        fixture.componentInstance.addStage();

        // Update pipeline
        fixture.componentInstance.pipeline = pipUpdated;

        // Detected pipeline updated
        fixture.componentInstance.ngDoCheck();


        expect(pipStore.addStage).toHaveBeenCalledWith('key', 'pip', s);
        expect(fixture.componentInstance.selectedStage).toBe(s);

    }));

    it('should update/delete a stage', fakeAsync( () => {

        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "pip", "stages": [] }'})));
        });

        // Create component
        let fixture = TestBed.createComponent(PipelineWorkflowComponent);
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
        this.params = Observable.of({key: 'key1', pipName: 'pip1'});
        this.queryParams = Observable.of({key: 'key1', appName: 'pip1'});
    }
}
