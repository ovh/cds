import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { CUSTOM_ELEMENTS_SCHEMA, Injector } from '@angular/core';
import { fakeAsync, getTestBed, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { ToasterService } from 'angular2-toaster-sgu';
import { ApplicationService } from 'app/service/application/application.service';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { HelpService } from 'app/service/help/help.service';
import { MonitoringService, RouterService } from 'app/service/services.module';
import { UserService } from 'app/service/user/user.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { NgxsStoreModule } from 'app/store/store.module';
import { Project } from '../../../model/project.model';
import { RepositoriesManager } from '../../../model/repositories.model';
import { EnvironmentService } from '../../../service/environment/environment.service';
import { NavbarService } from '../../../service/navbar/navbar.service';
import { PipelineService } from '../../../service/pipeline/pipeline.service';
import { ProjectService } from '../../../service/project/project.service';
import { ProjectStore } from '../../../service/project/project.store';
import { RepoManagerService } from '../../../service/repomanager/project.repomanager.service';
import { VariableService } from '../../../service/variable/variable.service';
import { SharedModule } from '../../shared.module';
import { RepoManagerFormComponent } from './repomanager.form.component';


describe('CDS: Project RepoManager Form Component', () => {

    let injector: Injector;
    let projectStore: ProjectStore;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                ApplicationService,
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                ToasterService,
                HelpService,
                TranslateService,
                TranslateParser,
                NavbarService,
                WorkflowService,
                WorkflowRunService,
                UserService,
                AuthenticationService,
                MonitoringService,
                RouterService
            ],
            imports: [
                SharedModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        }).compileComponents();
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
        let fixture = TestBed.createComponent(RepoManagerFormComponent);
        let component = fixture.debugElement.componentInstance;
        http.expectOne(((req: HttpRequest<any>) => req.url === '/repositories_manager')).flush(repoManMock);
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.button.disabled')).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        // Load project
        projectStore.getProjects('key1').subscribe(() => { });
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/key1')).flush(repoManMock);
    }));
});
