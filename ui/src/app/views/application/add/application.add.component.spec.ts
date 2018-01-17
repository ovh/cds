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
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {Project} from '../../../model/project.model';
import {ApplicationAddComponent} from './application.add.component';
import {Parameter} from '../../../model/parameter.model';
import {Application} from '../../../model/application.model';
import {Variable} from '../../../model/variable.model';
import {VariableService} from '../../../service/variable/variable.service';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import 'rxjs/add/observable/of';
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
