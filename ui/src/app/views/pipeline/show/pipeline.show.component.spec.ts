/* tslint:disable:no-unused-variable */
import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { fakeAsync, TestBed } from '@angular/core/testing';
import { ActivatedRoute, ActivatedRouteSnapshot } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddPipelineParameter, DeletePipelineParameter, FetchPipeline, UpdatePipelineParameter } from 'app/store/pipelines.action';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Parameter } from '../../../model/parameter.model';
import { Pipeline } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { KeyService } from '../../../service/keys/keys.service';
import { NavbarService } from '../../../service/navbar/navbar.service';
import { PipelineCoreService } from '../../../service/pipeline/pipeline.core.service';
import { PipelineService } from '../../../service/pipeline/pipeline.service';
import { PipelineStore } from '../../../service/pipeline/pipeline.store';
import { ProjectService } from '../../../service/project/project.service';
import { ProjectStore } from '../../../service/project/project.store';
import { ParameterEvent } from '../../../shared/parameter/parameter.event.model';
import { SharedModule } from '../../../shared/shared.module';
import { ToastService } from '../../../shared/toast/ToastService';
import { PipelineModule } from '../pipeline.module';
import { PipelineShowComponent } from './pipeline.show.component';
describe('CDS: Pipeline Show', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                KeyService,
                PipelineCoreService,
                PipelineService,
                PipelineStore,
                ProjectService,
                ProjectStore,
                NavbarService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                { provide: ToastService, useClass: MockToast },
                TranslateService,
                TranslateLoader,
                TranslateParser,
                AuthentificationStore
            ],
            imports: [
                PipelineModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });
    });

    it('should load component', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);

        let pipelineMock = new Pipeline();
        pipelineMock.name = 'pip1';

        // Create component
        let fixture = TestBed.createComponent(PipelineShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let store: Store = TestBed.get(Store);
        store.dispatch(new FetchPipeline({
            projectKey: 'key1',
            pipelineName: 'pip1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/key1/pipeline/pip1';
        })).flush(pipelineMock);

        fixture.componentInstance.ngOnInit();

        expect(fixture.componentInstance.selectedTab).toBe('workflow');
        expect(fixture.componentInstance.pipeline.name).toBe('pip1');
        expect(fixture.componentInstance.project.key).toBe('key1');

    }));

    it('should run add/update/delete parameters', fakeAsync(() => {

        // Create component
        let fixture = TestBed.createComponent(PipelineShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init data
        fixture.componentInstance.pipeline = new Pipeline();
        fixture.componentInstance.pipeline.name = 'pip1';

        fixture.componentInstance.project = new Project();
        fixture.componentInstance.project.key = 'key1';

        let param: Parameter = new Parameter();
        param.type = 'string';
        param.name = 'foo';
        param.value = 'bar';
        param.description = 'my description';

        // ADD

        let event: ParameterEvent = new ParameterEvent('add', param);
        let store: Store = TestBed.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => {
            return of(new Pipeline());
        });
        fixture.componentInstance.parameterEvent(event, true);
        expect(store.dispatch).toHaveBeenCalledWith(new AddPipelineParameter({
            projectKey: 'key1',
            pipelineName: 'pip1',
            parameter: param
        }));

        // Update

        event.type = 'update';
        fixture.componentInstance.parameterEvent(event, true);
        expect(store.dispatch).toHaveBeenCalledWith(new UpdatePipelineParameter({
            projectKey: 'key1',
            pipelineName: 'pip1',
            parameterName: 'foo',
            parameter: param
        }));

        // Delete
        event.type = 'delete';
        fixture.componentInstance.parameterEvent(event, true);
        expect(store.dispatch).toHaveBeenCalledWith(new DeletePipelineParameter({
            projectKey: 'key1',
            pipelineName: 'pip1',
            parameter: param
        }));
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = of({ key: 'key1', pipName: 'pip1' });
        this.queryParams = of({ key: 'key1', appName: 'pip1', tab: 'workflow' });
        this.snapshot = new ActivatedRouteSnapshot();
        this.snapshot.queryParams = {};

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project: project
        };
    }
}
