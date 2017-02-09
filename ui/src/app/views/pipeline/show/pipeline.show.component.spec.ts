/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {ActivatedRoute, ActivatedRouteSnapshot, Data} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {Observable} from 'rxjs';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {ProjectService} from '../../../service/project/project.service';
import {ProjectStore} from '../../../service/project/project.store';
import {PipelineService} from '../../../service/pipeline/pipeline.service';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {ToastService} from '../../../shared/toast/ToastService';
import {PipelineModule} from '../pipeline.module';
import {SharedModule} from '../../../shared/shared.module';
import {PipelineShowComponent} from './pipeline.show.component';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {GroupPermission, Group} from '../../../model/group.model';
import {Pipeline} from '../../../model/pipeline.model';
import {Project} from '../../../model/project.model';
import {Parameter} from '../../../model/parameter.model';
import {ParameterEvent} from '../../../shared/parameter/parameter.event.model';

describe('CDS: Pipeline Show', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: XHRBackend, useClass: MockBackend},
                PipelineService,
                PipelineStore,
                ProjectService,
                ProjectStore,
                {provide: ActivatedRoute, useClass: MockActivatedRoutes},
                {provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser,
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

    it('should load component', fakeAsync(() => {
        let call = 0;
        // Mock Http
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "prj1" }'})));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "pip1" }'})));
                    break;
            }

        });

        // Create component
        let fixture = TestBed.createComponent(PipelineShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let projStore: ProjectStore = injector.get(ProjectStore);
        projStore.getProjects('key1').subscribe(() => {
        }).unsubscribe();

        let pipStore: PipelineStore = injector.get(PipelineStore);
        pipStore.getPipelines('key1', 'pip1').subscribe(() => {
        }).unsubscribe();

        fixture.componentInstance.ngOnInit();

        expect(fixture.componentInstance.selectedTab).toBe('workflow');
        expect(fixture.componentInstance.pipeline.name).toBe('pip1');
        expect(fixture.componentInstance.project.key).toBe('key1');

    }));

    it('should run add/update/delete permission', fakeAsync(() => {
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
        let fixture = TestBed.createComponent(PipelineShowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init data
        fixture.componentInstance.pipeline = new Pipeline();
        fixture.componentInstance.pipeline.name = 'pip1';

        fixture.componentInstance.project = new Project();
        fixture.componentInstance.project.key = 'key1';

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 7;
        gp.group = new Group();
        gp.group.name = 'grp1';

        let pipStore: PipelineStore = injector.get(PipelineStore);
        spyOn(pipStore, 'addPermission').and.callFake(() => {
            return Observable.of(new Pipeline());
        });

        // ADD

        let groupEvent: PermissionEvent = new PermissionEvent('add', gp);
        fixture.componentInstance.groupEvent(groupEvent, true);
        expect(pipStore.addPermission).toHaveBeenCalledWith('key1', 'pip1', gp);

        // Update

        groupEvent.type = 'update';
        spyOn(pipStore, 'updatePermission').and.callFake(() => {
            return Observable.of(new Pipeline());
        });
        fixture.componentInstance.groupEvent(groupEvent, true);
        expect(pipStore.updatePermission).toHaveBeenCalledWith('key1', 'pip1', gp);

        // Delete
        groupEvent.type = 'delete';
        spyOn(pipStore, 'removePermission').and.callFake(() => {
            return Observable.of(new Pipeline());
        });
        fixture.componentInstance.groupEvent(groupEvent, true);
        expect(pipStore.removePermission).toHaveBeenCalledWith('key1', 'pip1', gp);
    }));

    it('should run add/update/delete parameters', fakeAsync(() => {
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
        let pipStore: PipelineStore = injector.get(PipelineStore);
        spyOn(pipStore, 'addParameter').and.callFake(() => {
            return Observable.of(new Pipeline());
        });
        fixture.componentInstance.parameterEvent(event, true);
        expect(pipStore.addParameter).toHaveBeenCalledWith('key1', 'pip1', param);

        // Update

        event.type = 'update';
        spyOn(pipStore, 'updateParameter').and.callFake(() => {
            return Observable.of(new Pipeline());
        });
        fixture.componentInstance.parameterEvent(event, true);
        expect(pipStore.updateParameter).toHaveBeenCalledWith('key1', 'pip1', param);

        // Delete
        event.type = 'delete';
        spyOn(pipStore, 'removeParameter').and.callFake(() => {
            return Observable.of(new Pipeline());
        });
        fixture.componentInstance.parameterEvent(event, true);
        expect(pipStore.removeParameter).toHaveBeenCalledWith('key1', 'pip1', param);
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
        this.queryParams = Observable.of({key: 'key1', appName: 'pip1', tab: 'workflow'});
        this.snapshot = new ActivatedRouteSnapshot();
        this.snapshot.queryParams = {};

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project: project
        };
    }
}
