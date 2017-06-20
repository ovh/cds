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
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {ApplicationStore} from '../../service/application/application.store';
import {ApplicationService} from '../../service/application/application.service';
import {Project} from '../../model/project.model';
import {LanguageStore} from '../../service/language/language.store';
import {RouterService} from '../../service/router/router.service';
import {WarningStore} from '../../service/warning/warning.store';

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
                RouterService,
                WarningStore,
                AuthentificationStore,
                ApplicationStore,
                ApplicationService,
                TranslateParser,
                LanguageStore
            ],
            imports: [
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

        injector = getTestBed();
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        projectStore = undefined;
    });


    it('should select a project + rename project event', fakeAsync(inject([XHRBackend], (backend: MockBackend) => {
        let call = 0;
        let nameUpdated = 'prj1Updated';
        // Mock Http login request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `[
                        { "key": "key1", "name": "prj1" },
                        { "key": "key2", "name": "prj2" }
                    ]`
                    })));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "' + nameUpdated + '" }'})));
                    break;
            }

        });

        // Create loginComponent
        let fixture = TestBed.createComponent(NavbarComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();
        expect(backend.connectionsArray.length).toBe(1, 'Must have call getProjects');

    })));
});
