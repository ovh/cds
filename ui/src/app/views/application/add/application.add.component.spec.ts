/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed} from '@angular/core/testing';
import {Router, ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';
import {EnvironmentService} from '../../../service/environment/environment.service';
import {ApplicationStore} from '../../../service/application/application.store';
import {ApplicationService} from '../../../service/application/application.service';
import {PipelineService} from '../../../service/pipeline/pipeline.service';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../../../shared/shared.module';
import {ServicesModule} from '../../../service/services.module';
import {Observable} from 'rxjs/Observable';
import {Injector} from '@angular/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {ProjectStore} from '../../../service/project/project.store';
import {ProjectService} from '../../../service/project/project.service';
import {ApplicationModule} from '../application.module';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {Project} from '../../../model/project.model';
import {ApplicationAddComponent} from './application.add.component';
import {ApplicationTemplateService} from '../../../service/application/application.template.service';
import {Template, ApplyTemplateRequest} from '../../../model/template.model';
import {Parameter} from '../../../model/parameter.model';
import {Application} from '../../../model/application.model';
import {Variable} from '../../../model/variable.model';
import {VariableService} from '../../../service/variable/variable.service';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Application Add Component', () => {

    let injector: Injector;
    let appStore: ApplicationStore;
    let router: Router;
    let prjStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                AuthentificationStore,
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                EnvironmentService,
                PipelineService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: Router, useClass: MockRouter},
                { provide: ToastService, useClass: MockToast},
                TranslateService,
                TranslateLoader,
                TranslateParser,
                ApplicationTemplateService,
                VariableService
            ],
            imports : [
                ApplicationModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                ServicesModule,
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
        appStore = injector.get(ApplicationStore);
        router = injector.get(Router);
        prjStore = injector.get(ProjectStore);
    });

    afterEach(() => {
        injector = undefined;
        appStore = undefined;
        router = undefined;
        prjStore = undefined;
    });

    it('should create an empty application', fakeAsync( () => {

        // Create component
        let fixture = TestBed.createComponent(ApplicationAddComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project: Project = new Project();
        project.key = 'key1';

        let templates = new Array<Template>();
        let t1 = new Template();
        t1.name = 'Void';
        templates.push(t1);
        let t2 = new Template();
        t2.name = 'GoBuild';
        templates.push(t2);
        fixture.componentInstance.templates = templates;

        fixture.componentInstance.selectedName = 'myApp';
        fixture.componentInstance.updateSelection('empty');

        expect(fixture.componentInstance.selectedTemplate.name).toBe('Void');

        spyOn(appStore, 'applyTemplate').and.callFake( () => {
            return Observable.of(project);
        });

        fixture.componentInstance.createApplication();
        let addAppRequest: ApplyTemplateRequest = new ApplyTemplateRequest();
        addAppRequest.name = 'myApp';
        addAppRequest.template = 'Void';
        expect(appStore.applyTemplate).toHaveBeenCalledWith(project.key, addAppRequest);
    }));

    it('should create an application from template', fakeAsync( () => {

        // Create component
        let fixture = TestBed.createComponent(ApplicationAddComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project: Project = new Project();
        project.key = 'key1';

        let templates = new Array<Template>();
        let t1 = new Template();
        t1.name = 'Void';
        templates.push(t1);
        let t2 = new Template();
        t2.name = 'GoBuild';
        t2.params = new Array<Parameter>();
        let p = new Parameter();
        p.name = 'param1';
        p.type = 'number';
        p.value = '1';
        t2.params.push(p);
        templates.push(t2);
        fixture.componentInstance.templates = templates;

        fixture.componentInstance.selectedName = 'myApp';
        fixture.componentInstance.typeofCreation = 'template';
        fixture.componentInstance.updateSelection('template');
        fixture.componentInstance.updateSelectedTemplateToUse('GoBuild');

        expect(fixture.componentInstance.selectedTemplate.name).toBe('GoBuild');
        expect(fixture.componentInstance.parameters.length).toBe(1);

        fixture.componentInstance.parameters[0].value = '2';

        spyOn(appStore, 'applyTemplate').and.callFake( () => {
            return Observable.of(project);
        });

        fixture.componentInstance.createApplication();
        let addAppRequest: ApplyTemplateRequest = new ApplyTemplateRequest();
        addAppRequest.name = 'myApp';
        addAppRequest.template = 'GoBuild';
        addAppRequest.template_params = fixture.componentInstance.parameters;

        expect(appStore.applyTemplate).toHaveBeenCalledWith(project.key, addAppRequest);
    }));

    it('should clone an application', fakeAsync( () => {

        // Create component
        let fixture = TestBed.createComponent(ApplicationAddComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project: Project = new Project();
        project.key = 'key1';
        project.applications = new Array<Application>();
        let app = new Application();
        app.name = 'appToClone';
        app.variables = new Array<Variable>();
        let v = new Variable();
        v.name = 'var1';
        v.type = 'password';
        v.value = 'value';
        app.variables.push(v);
        project.applications.push(app);

        spyOn(appStore, 'getApplicationResolver').and.callFake(() => {
            return Observable.of(app);
        });

        fixture.componentInstance.project = project;
        fixture.componentInstance.selectedName = 'myApp';
        fixture.componentInstance.typeofCreation = 'clone';
        fixture.componentInstance.updateSelection('clone');

        expect(fixture.componentInstance.selectedApplication.name).toBe('appToClone');
        expect(fixture.componentInstance.variables.length).toBe(1);
        expect(fixture.componentInstance.variables[0].value).toBe('');


        spyOn(appStore, 'cloneApplication').and.callFake( () => {
            return Observable.of(app);
        });

        fixture.componentInstance.createApplication();
        let appRequest = new Application();
        appRequest.name = 'myApp';
        appRequest.variables = fixture.componentInstance.variables;
        expect(appStore.cloneApplication).toHaveBeenCalledWith(project.key, 'appToClone', appRequest);
    }));

    it('should display error for unalloawed application name pattern', fakeAsync( () => {

        // Create component
        let fixture = TestBed.createComponent(ApplicationAddComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.selectedName = 'ééé##';

        fixture.componentInstance.createApplication();

        expect(fixture.componentInstance.appPatternError).toBeTruthy();
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
        this.params = Observable.of({key: 'key1', appName: 'app1'});
        this.queryParams = Observable.of({key: 'key1', appName: 'app1'});

        this.snapshot = new ActivatedRouteSnapshot();

        let project = new Project();
        project.key = 'key1';
        this.snapshot.data = {
            project: project
        };

        this.data = Observable.of({ project: project });
    }
}
