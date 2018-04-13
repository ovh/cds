/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import { TranslateService, TranslateLoader} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {ProjectRepoManagerFormComponent} from './project.repomanager.form.component';
import {RepoManagerService} from '../../../../../../service/repomanager/project.repomanager.service';
import {ProjectStore} from '../../../../../../service/project/project.store';
import {ProjectService} from '../../../../../../service/project/project.service';
import {PipelineService} from '../../../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../../../service/environment/environment.service';
import {VariableService} from '../../../../../../service/variable/variable.service';
import {SharedModule} from '../../../../../../shared/shared.module';
import {ToasterService} from 'angular2-toaster/angular2-toaster';
import {Project} from '../../../../../../model/project.model';
import {TranslateParser} from '@ngx-translate/core';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {RepositoriesManager} from '../../../../../../model/repositories.model';
import {HttpRequest} from '@angular/common/http';

describe('CDS: Project RepoManager Form Component', () => {

    let injector: Injector;
    let projectStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
                ProjectRepoManagerFormComponent
            ],
            providers: [
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                ToasterService,
                TranslateService,
                TranslateParser
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        projectStore = undefined;
    });


    it('Add new repo manager', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);

        let repoManMock = new Array<RepositoriesManager>();
        let stash = new RepositoriesManager();
        stash.name = 'stash.com';
        let github = new RepositoriesManager();
        github.name = 'github.com';
        repoManMock.push(stash, github);

        let projectMock = new Project();
        projectMock.name = 'prj1';
        projectMock.key = 'key1';
        projectMock.last_modified = '0';
        projectMock.vcs_servers = [];

        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectRepoManagerFormComponent);
        let component = fixture.debugElement.componentInstance;
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/repositories_manager';
        })).flush(repoManMock);
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.button.disabled')).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        // Load project
        projectStore.getProjects('key1').subscribe(() => {});
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/key1';
        })).flush(repoManMock);

    }));
});
