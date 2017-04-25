/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {ProjectStore} from '../../../../../service/project/project.store';
import {ProjectService} from '../../../../../service/project/project.service';
import {ProjectModule} from '../../../project.module';
import {Project} from '../../../../../model/project.model';
import {SharedModule} from '../../../../../shared/shared.module';
import {Environment} from '../../../../../model/environment.model';
import {ProjectEnvironmentListComponent} from './environment.list.component';
import {ToasterService} from 'angular2-toaster';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {VariableService} from '../../../../../service/variable/variable.service';
import {ActivatedRoute, Router} from '@angular/router';
import {Observable} from 'rxjs/Rx';

describe('CDS: Environment List Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ProjectStore,
                ProjectService,
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                ToasterService,
                ToastService,
                TranslateLoader,
                TranslateParser,
                VariableService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes},
                { provide: Router, useClass: MockRouter},
            ],
            imports : [
                ProjectModule,
                SharedModule,
                RouterTestingModule.withRoutes([
                    { path: 'project/:key', component: ProjectEnvironmentListComponent },
                ])
            ]
        });

        this.injector = getTestBed();
    });

    afterEach(() => {
        this.injector = undefined;
    });

    it('should load component', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ProjectEnvironmentListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();


        let project = new Project();
        project.key = 'key1';

        let envs = new Array<Environment>();
        let e = new Environment();
        e.name = 'prod';
        envs.push(e);
        project.environments = envs;

        fixture.componentInstance.project = project;
        fixture.componentInstance.ngOnInit();
    }));
});

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();

        this.queryParams = Observable.of({envName: 'prod'});
    }
}

class MockRouter {
    public navigate() {
    }
}


