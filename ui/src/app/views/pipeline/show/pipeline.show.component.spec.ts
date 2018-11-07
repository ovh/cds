/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync} from '@angular/core/testing';
import {ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {of} from 'rxjs';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
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
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import 'rxjs/add/observable/of';
import {NavbarService} from '../../../service/navbar/navbar.service';
import {KeyService} from '../../../service/keys/keys.service';
import {PipelineCoreService} from '../../../service/pipeline/pipeline.core.service';
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
                ApplicationPipelineService,
                {provide: ActivatedRoute, useClass: MockActivatedRoutes},
                {provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser,
                AuthentificationStore
            ],
            imports: [
                PipelineModule,
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

        let pipStore: PipelineStore = TestBed.get(PipelineStore);
        pipStore.getPipelines('key1', 'pip1').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/key1/pipeline/pip1';
        })).flush(pipelineMock);

        fixture.componentInstance.ngOnInit();

        expect(fixture.componentInstance.selectedTab).toBe('workflow');
        expect(fixture.componentInstance.pipeline.name).toBe('pip1');
        expect(fixture.componentInstance.project.key).toBe('key1');

    }));

    it('should run add/update/delete permission', fakeAsync(() => {

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

        let pipStore: PipelineStore = TestBed.get(PipelineStore);
        spyOn(pipStore, 'addPermission').and.callFake(() => {
            return of(new Pipeline());
        });

        // ADD

        let groupEvent: PermissionEvent = new PermissionEvent('add', gp);
        fixture.componentInstance.groupEvent(groupEvent, true);
        expect(pipStore.addPermission).toHaveBeenCalledWith('key1', 'pip1', gp);

        // Update

        groupEvent.type = 'update';
        spyOn(pipStore, 'updatePermission').and.callFake(() => {
            return of(new Pipeline());
        });
        fixture.componentInstance.groupEvent(groupEvent, true);
        expect(pipStore.updatePermission).toHaveBeenCalledWith('key1', 'pip1', gp);

        // Delete
        groupEvent.type = 'delete';
        spyOn(pipStore, 'removePermission').and.callFake(() => {
            return of(new Pipeline());
        });
        fixture.componentInstance.groupEvent(groupEvent, true);
        expect(pipStore.removePermission).toHaveBeenCalledWith('key1', 'pip1', gp);
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
        let pipStore: PipelineStore = TestBed.get(PipelineStore);
        spyOn(pipStore, 'addParameter').and.callFake(() => {
            return of(new Pipeline());
        });
        fixture.componentInstance.parameterEvent(event, true);
        expect(pipStore.addParameter).toHaveBeenCalledWith('key1', 'pip1', param);

        // Update

        event.type = 'update';
        spyOn(pipStore, 'updateParameter').and.callFake(() => {
            return of(new Pipeline());
        });
        fixture.componentInstance.parameterEvent(event, true);
        expect(pipStore.updateParameter).toHaveBeenCalledWith('key1', 'pip1', param);

        // Delete
        event.type = 'delete';
        spyOn(pipStore, 'removeParameter').and.callFake(() => {
            return of(new Pipeline());
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
        this.params = of({key: 'key1', pipName: 'pip1'});
        this.queryParams = of({key: 'key1', appName: 'pip1', tab: 'workflow'});
        this.snapshot = new ActivatedRouteSnapshot();
        this.snapshot.queryParams = {};

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project: project
        };
    }
}
