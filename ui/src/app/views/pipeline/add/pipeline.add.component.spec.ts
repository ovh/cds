/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Router, ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../../../shared/shared.module';
import {of} from 'rxjs';
import {Injector} from '@angular/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {ProjectStore} from '../../../service/project/project.store';
import {ProjectService} from '../../../service/project/project.service';
import {PipelineService} from '../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../service/environment/environment.service';
import {VariableService} from '../../../service/variable/variable.service';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {Project} from '../../../model/project.model';
import {Application} from '../../../model/application.model';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {PipelineModule} from '../pipeline.module';
import {PipelineAddComponent} from './pipeline.add.component';
import {Pipeline} from '../../../model/pipeline.model';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import 'rxjs/add/observable/of';
import {NavbarService} from '../../../service/navbar/navbar.service';
describe('CDS: Pipeline Add Component', () => {

    let injector: Injector;
    let pipStore: PipelineStore;
    let backend: MockBackend;
    let router: Router;
    let prjStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                AuthentificationStore,
                ProjectStore,
                ProjectService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: Router, useClass: MockRouter},
                { provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser,
                PipelineStore,
                NavbarService,
                PipelineService,
                EnvironmentService,
                VariableService
            ],
            imports : [
                PipelineModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(MockBackend);
        pipStore = injector.get(PipelineStore);
        router = injector.get(Router);
        prjStore = injector.get(ProjectStore);
    });

    afterEach(() => {
        injector = undefined;
        pipStore = undefined;
        backend = undefined;
        router = undefined;
        prjStore = undefined;
    });

    it('should create an empty pipeline', fakeAsync( () => {

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
        fixture.componentInstance.newPipeline.type = 'build';

        spyOn(pipStore, 'createPipeline').and.callFake( () => {
            return of(fixture.componentInstance.newPipeline);
        });

        fixture.componentInstance.createPipeline();
        expect(pipStore.createPipeline).toHaveBeenCalledWith(project.key, fixture.componentInstance.newPipeline);

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
        this.params = of({key: 'key1', appName: 'app1'});
        this.queryParams = of({key: 'key1', appName: 'app1'});

        this.snapshot = new ActivatedRouteSnapshot();

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project: project
        };

        this.data = of({ project: project });
    }
}
