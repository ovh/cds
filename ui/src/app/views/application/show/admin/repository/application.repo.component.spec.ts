import { APP_BASE_HREF } from '@angular/common';
import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { Component, Injector, NO_ERRORS_SCHEMA } from '@angular/core';
import { fakeAsync, getTestBed, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Project } from 'app/model/project.model';
import { RepositoriesManager } from 'app/model/repositories.model';
import { ApplicationService } from 'app/service/application/application.service';
import { ApplicationStore } from 'app/service/application/application.store';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { KeyService } from 'app/service/keys/keys.service';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { MonitoringService, RouterService, ThemeStore, UserService } from 'app/service/services.module';
import { VariableService } from 'app/service/variable/variable.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { SharedModule } from 'app/shared/shared.module';
import { ToastService } from 'app/shared/toast/ToastService';
import { NgxsStoreModule } from 'app/store/store.module';
import { ApplicationModule } from 'app/views/application/application.module';
import { ProjectModule } from 'app/views/project/project.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { ApplicationRepositoryComponent } from './application.repo.component';

@Component({
    template: ''
})
class DummyComponent {
}

describe('CDS: Application Repo Component', () => {
    let injector: Injector;
    let toast: ToastService;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                DummyComponent
            ],
            providers: [
                { provide: ApplicationStore, useClass: MockStore },
                ApplicationService,
                KeyService,
                ProjectStore,
                NavbarService,
                ProjectService,
                MonitoringService,
                PipelineService,
                EnvironmentService,
                VariableService,
                { provide: ToastService, useClass: MockToast },
                TranslateLoader,
                TranslateService,
                TranslateParser,
                RepoManagerService,
                Store,
                { provide: APP_BASE_HREF, useValue: '/' },
                ThemeStore,
                RouterService,
                WorkflowRunService,
                WorkflowService,
                UserService,
                AuthenticationService
            ],
            imports: [
                RouterTestingModule.withRoutes([
                    { path: 'project/:key', component: DummyComponent }
                ]),
                ProjectModule,
                ApplicationModule,
                SharedModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ],
            schemas: [NO_ERRORS_SCHEMA]
        }).compileComponents();


        injector = getTestBed();
        toast = injector.get(ToastService);
    });

    afterEach(() => {
        injector = undefined;
        toast = undefined;
    });

    it('Load component + select repository', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let mockResponse = [
            { name: 'repo1', fullname: 'frepo1' },
            { name: 'repo2', fullname: 'frepo2' },
            { name: 'repo3', fullname: 'frepo3' },
            { name: 'repo4', fullname: 'frepo4' },
            { name: 'repo5', fullname: 'frepo5' }
        ];

        let fixture = TestBed.createComponent(ApplicationRepositoryComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let app: Application = new Application();
        app.name = 'app';
        let p: Project = new Project();
        p.key = 'key1';
        p.name = 'proj1';

        let repoMan: RepositoriesManager = { name: 'RepoManager' };
        p.vcs_servers = new Array<RepositoriesManager>();
        p.vcs_servers.push(repoMan);

        fixture.componentInstance.application = app;
        fixture.componentInstance.project = p;

        fixture.componentInstance.ngOnInit();
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/key1/repositories_manager/RepoManager/repos')).flush(mockResponse);

        expect(fixture.componentInstance.selectedRepoManager).toBe('RepoManager');
        expect(fixture.componentInstance.repos.length).toBe(5, 'Must have 5 repositories in list');

        // Select repo + link
        fixture.componentInstance.selectedRepo = 'frepo3';

        fixture.detectChanges();
        tick(50);

        spyOn(toast, 'success');

        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('button[name="addrepobtn"]').click();
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/key1/repositories_manager/RepoManager/application/app/attach')).flush({
            name: 'app',
            vcs_server: 'RepoManager',
            repository_fullname: 'frepo3'
        });
        fixture.detectChanges();
        tick(100);

        expect(toast.success).toHaveBeenCalledTimes(1);

        tick(100);
        fixture.componentInstance.application.vcs_server = repoMan.name;
        fixture.componentInstance.application.repository_fullname = 'frepo3';

        tick(100);

        // Detach repo
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges();
        tick(50);
        compiled.querySelector('.ui.red.button.active').click();
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/key1/repositories_manager/RepoManager/application/app/detach')).flush({
            name: 'app',
        });
        tick(100);
        expect(toast.success).toHaveBeenCalledTimes(2);
    }));
});

class MockRouter {
    public navigate() {
    }
}

class MockStore {
    constructor() { }

    connectRepository(key: string, currentName: string, repoManName: string, repoFullname: string) {
        return of({ name: 'app' });
    }
    removeRepository(key: string, currentName: string, repoManName: string, repoFullname: string) {
        return of({ name: 'app' });
    }
}

class MockToast {
    success(title: string, msg: string) {

    }
}
