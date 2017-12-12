/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {ApplicationStore} from '../../../../../../service/application/application.store';
import {ApplicationService} from '../../../../../../service/application/application.service';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ProjectService} from '../../../../../../service/project/project.service';
import {PipelineService} from '../../../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../../../service/environment/environment.service';
import {VariableService} from '../../../../../../service/variable/variable.service';
import {AuthentificationStore} from '../../../../../../service/auth/authentification.store';
import {SharedModule} from '../../../../../../shared/shared.module';
import {ApplicationModule} from '../../../../application.module';
import {Project} from '../../../../../../model/project.model';
import {Application, ApplicationPipeline} from '../../../../../../model/application.model';
import {Pipeline} from '../../../../../../model/pipeline.model';
import {ApplicationPipelineLinkComponent} from './pipeline.link.component';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {ToasterService} from 'angular2-toaster';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Application pipeline link', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: XHRBackend, useClass: MockBackend},
                AuthentificationStore,
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                ToastService,
                ToasterService,
                PipelineService,
                EnvironmentService,
                VariableService
            ],
            imports: [
                ApplicationModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });

    it('should check that only 1 pipeline can be attached', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationPipelineLinkComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init component Input
        let p = new Project();
        p.key = 'key1';
        fixture.componentInstance.project = p;

        let pip1 = new Pipeline();
        pip1.name = 'pip1';
        let pip2 = new Pipeline();
        pip2.name = 'pip2';
        p.pipelines = [pip1, pip2];

        let a = new Application();
        a.pipelines = new Array<ApplicationPipeline>();
        let ap1 = new ApplicationPipeline();
        ap1.pipeline = pip1;
        a.pipelines.push(ap1);
        fixture.componentInstance.application = a;

        expect(fixture.componentInstance.getLinkablePipelines().length).toBe(1);
        expect(fixture.componentInstance.getLinkablePipelines()[0].name).toBe('pip2');
    }));
});
