/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, fakeAsync, inject} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {NavbarComponent} from './navbar.component';
import {SharedModule} from '../../shared/shared.module';
import {ProjectStore} from '../../service/project/project.store';
import {ProjectService} from '../../service/project/project.service';
import {EnvironmentService} from '../../service/environment/environment.service';
import {PipelineService} from '../../service/pipeline/pipeline.service';
import {VariableService} from '../../service/variable/variable.service';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {ApplicationStore} from '../../service/application/application.store';
import {ApplicationService} from '../../service/application/application.service';
import {Project} from '../../model/project.model';
import {LanguageStore} from '../../service/language/language.store';
import {RouterService} from '../../service/router/router.service';
import {WarningStore} from '../../service/warning/warning.store';
import {WarningService} from '../../service/warning/warning.service';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import {User} from '../../model/user.model';

describe('CDS: Navbar Component', () => {

    let injector: Injector;
    let projectStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
                NavbarComponent
            ],
            providers: [
                TranslateService,
                {provide: XHRBackend, useClass: MockBackend},
                TranslateLoader,
                ProjectStore,
                ProjectService,
                EnvironmentService,
                PipelineService,
                VariableService,
                RouterService,
                WarningStore,
                WarningService,
                AuthentificationStore,
                ApplicationStore,
                ApplicationService,
                TranslateParser,
                LanguageStore
            ],
            imports: [
                SharedModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        projectStore = undefined;
    });


    it('should select a project + rename project event', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);
        const authStore = TestBed.get(AuthentificationStore);

        let projects = new Array<Project>();
        let p1 = new Project();
        p1.key = 'key1';
        p1.name = 'prj1';
        let p2 = new Project();
        p2.key = 'key1';
        p2.name = 'prj1';
        projects.push(p1, p2);

        let fixture = TestBed.createComponent(NavbarComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        authStore.addUser(new User(), false);
        fixture.componentInstance.ngOnInit();
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(projects);

        http.verify();
    }));
});
