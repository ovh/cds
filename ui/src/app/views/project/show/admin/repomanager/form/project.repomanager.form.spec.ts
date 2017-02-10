/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import { TranslateService, TranslateLoader} from 'ng2-translate/ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {ProjectRepoManagerFormComponent} from './project.repomanager.form.component';
import {RepoManagerService} from '../../../../../../service/repomanager/project.repomanager.service';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ProjectService} from '../../../../../../service/project/project.service';
import {SharedModule} from '../../../../../../shared/shared.module';
import {ToasterService} from 'angular2-toaster/angular2-toaster';
import {Project} from '../../../../../../model/project.model';
import {TranslateParser} from 'ng2-translate';

describe('CDS: Project RepoManager Form Component', () => {

    let injector: Injector;
    let backend: MockBackend;
    let projectStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
                ProjectRepoManagerFormComponent
            ],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                ToasterService,
                TranslateService,
                TranslateParser
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([]),

            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        backend = injector.get(XHRBackend);
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        projectStore = undefined;
    });


    it('Add new repo manager', fakeAsync( () => {
        let call = 0;
        // Mock Http login request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1:
                    connection.mockRespond(new Response(new ResponseOptions({ body :
                        '[ { "name": "stash.com" }, { "name": "github.com" } ]'
                    })));
                    break;
                case 2:
                    connection.mockRespond(new Response(new ResponseOptions({ body : `{ 
                        "key": "key1",
                        "name": "prj1",
                        "last_modified": 0,
                        "repositories_manager" : []
                    }`})));
                    break;
                case 3:
                    connection.mockRespond(new Response(new ResponseOptions({ body : `{ 
                        "project_key": "key1",  
                        "last_modified" : "123", 
                        "repositories_manager" : 
                        "stash.com", 
                        "url" : "ff/.c",
                        "request_token": "mytoken"
                    }`
                    })));
                    break;
            }
        });


        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectRepoManagerFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.button.disabled')).toBeTruthy();

        expect(call).toBe(1, 'List of repo managers must have been loaded');

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;

        // Load project
        projectStore.getProjects('key1').subscribe(() => {});
        expect(call).toBe(2, 'Get project mast have been called');

        // Add a repo
        fixture.componentInstance.selectedRepo = 'stash.com';

        let p: Project = new Project();
        p.key = 'key1';
        fixture.componentInstance.project = p;

        /*   MODAL TESTING KO
        compiled.querySelector('button[name="addrepo"]').click();
        expect(call).toBe(3, "Connect repo must have been called");

        // Check project update
        projectStore.getProject("key1").subscribe(p => {
            expect(p.last_modified).toBe(123, "Project in cache must have been updated");
        }).unsubscribe();
        expect(call).toBe(3, "Project must have been caught in cache");


        // Enter validation code
        let inputCode = compiled.querySelector('input[name="verifiercode"]');
        inputCode.value="mycode";
        inputCode.dispatchEvent(new Event('input'));

        expect(fixture.componentInstance.validationToken).toBe("mycode");

        // send validation
        compiled.querySelector('button[name="validationbtn"]').click();
        expect(call).toBe(4, "Validation request must have been sent");
        */
    }));
});

