/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {XHRBackend} from '@angular/http';
import {MockBackend} from '@angular/http/testing';

import {SharedModule} from '../../../../../shared/shared.module';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ApplicationService} from '../../../../../service/application/application.service';
import {ProjectStore} from '../../../../../service/project/project.store';
import {ProjectService} from '../../../../../service/project/project.service';
import {ApplicationTriggerComponent} from './trigger.component';
import {Project} from '../../../../../model/project.model';
import {Application, ApplicationPipeline} from '../../../../../model/application.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {Trigger} from '../../../../../model/trigger.model';
import {ApplicationModule} from '../../../application.module';
import {Map} from 'immutable';
import {Observable} from 'rxjs/Observable';
import {Parameter} from '../../../../../model/parameter.model';
import {PrerequisiteEvent} from '../../../../../shared/prerequisites/prerequisite.event.model';
import {Prerequisite} from '../../../../../model/prerequisite.model';
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
                TranslateService,
                TranslateLoader,
                TranslateParser,
                ApplicationStore, ApplicationService,
                ProjectStore, ProjectService
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

    it('should create a new trigger', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationTriggerComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init Input Data
        let p: Project = new Project();
        p.key = 'key1';
        p.name = 'projectName';

        let a: Application = new Application();
        a.name = 'app1';

        let pip: Pipeline = new Pipeline();
        pip.name = 'pip1';

        let pip2 = new Pipeline();
        pip2.name = 'pip2';
        pip2.parameters = new Array<Parameter>();
        let param1 = new Parameter();
        param1.name = 'param1';
        pip2.parameters.push(param1);
        let param2 = new Parameter();
        param2.name = 'param2';
        pip2.parameters.push(param2);

        let trigger = new Trigger();
        trigger.src_application = a;
        trigger.src_pipeline = pip;
        trigger.src_project = p;
        trigger.dest_application = new Application();
        trigger.dest_pipeline = new Pipeline();
        trigger.dest_project = p;

        fixture.componentInstance.project = p;
        fixture.componentInstance.mode = 'create';
        fixture.componentInstance.trigger = trigger;

        expect(fixture.componentInstance.refPrerequisites.length).toBe(1);
        expect(fixture.componentInstance.appPipelines).toBeFalsy();

        fixture.detectChanges();
        tick(250);


        // select dest app
        fixture.componentInstance.trigger.dest_application = a;
        let appStore: ApplicationStore = injector.get(ApplicationStore);
        spyOn(appStore, 'getApplications').and.callFake(() => {
            let mapApp: Map<string, Application> = Map<string, Application>();
            let app: Application = new Application();
            app.name = 'app1';

            let pips = new Array<ApplicationPipeline>();
            let appPip1 = new ApplicationPipeline();
            appPip1.pipeline = pip;
            pips.push(appPip1);

            let appPip2 = new ApplicationPipeline();
            appPip2.pipeline = pip2;
            pips.push(appPip2);

            app.pipelines = pips;

            return Observable.of(mapApp.set('key1-app1', app));
        });
        fixture.componentInstance.updatePipelineList();
        expect(fixture.componentInstance.appPipelines.length).toBe(2);

        // select dest pip
        fixture.componentInstance.trigger.dest_pipeline = pip2;
        fixture.componentInstance.updateDestPipeline();
        expect(fixture.componentInstance.refPrerequisites.length).toBe(3);
        expect(fixture.componentInstance.trigger.parameters.length).toBe(2);


        // False because previous operation cannot be executed to edit a trigger
        expect(fixture.componentInstance.trigger.hasChanged).toBeFalsy();

        // add prerequisite
        let prerequisite = new Prerequisite();
        prerequisite.parameter = 'git.branch';
        prerequisite.expected_value = 'master';
        let event = new PrerequisiteEvent('add', prerequisite);
        fixture.componentInstance.prerequisiteEvent(event);
        // twice
        fixture.componentInstance.prerequisiteEvent(event);

        expect(fixture.componentInstance.trigger.prerequisites.length).toBe(1);
        expect(fixture.componentInstance.trigger.hasChanged).toBeTruthy();

        // delete prerequisite
        event.type = 'delete';
        fixture.componentInstance.prerequisiteEvent(event);
        expect(fixture.componentInstance.trigger.prerequisites.length).toBe(0);
    }));
});
