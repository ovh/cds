/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {TranslateService, TranslateParser, TranslateLoader} from 'ng2-translate';
import {ApplicationPipelineBuildComponent} from './pipeline.build.component';
import {ApplicationRunModule} from './application.run.module';
import {SharedModule} from '../../shared/shared.module';
import {ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Observable';
import {Project} from '../../model/project.model';
import {Application} from '../../model/application.model';
import {Pipeline} from '../../model/pipeline.model';
import {ApplicationPipelineService} from '../../service/application/pipeline/application.pipeline.service';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {RouterService} from '../../service/router/router.service';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Application Run Component', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                {provide: ActivatedRoute, useClass: MockActivatedRoutes},
                TranslateService,
                TranslateParser,
                ApplicationPipelineService,
                AuthentificationStore,
                RouterService,
                TranslateLoader,
                TranslateService,
                TranslateParser
            ],
            imports : [
                ApplicationRunModule,
                SharedModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });


    it('should load component', async( () => {
        let fixture = TestBed.createComponent(ApplicationPipelineBuildComponent);
        let app = fixture.debugElement.componentInstance;
        expect(app).toBeTruthy();
    }));
});

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({buildNumber: '123'});

        this.snapshot = new ActivatedRouteSnapshot();
        Object.defineProperty(this.snapshot, 'children', []);
        this.snapshot.queryParams = { envName: 'NoEnv'};

        this.queryParams = Observable.of({ envName: 'NoEnv'});

        let project = new Project();
        project.key = 'key1';

        let application = new Application();
        application.name = 'app1';

        let pipeline = new Pipeline();
        pipeline.name = 'pipName';

        this.data = Observable.of({ project: project, application: application, pipeline: pipeline });


    }
}
