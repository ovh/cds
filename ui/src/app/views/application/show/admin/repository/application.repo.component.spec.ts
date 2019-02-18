/* tslint:disable:no-unused-variable */

import { APP_BASE_HREF } from '@angular/common';
import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { Component, Injector, NO_ERRORS_SCHEMA } from '@angular/core';
import { fakeAsync, getTestBed, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Application } from '../../../../../model/application.model';
import { Project } from '../../../../../model/project.model';
import { RepositoriesManager } from '../../../../../model/repositories.model';
import { ApplicationService } from '../../../../../service/application/application.service';
import { ApplicationStore } from '../../../../../service/application/application.store';
import { EnvironmentService } from '../../../../../service/environment/environment.service';
import { KeyService } from '../../../../../service/keys/keys.service';
import { NavbarService } from '../../../../../service/navbar/navbar.service';
import { PipelineService } from '../../../../../service/pipeline/pipeline.service';
import { ProjectService } from '../../../../../service/project/project.service';
import { ProjectStore } from '../../../../../service/project/project.store';
import { RepoManagerService } from '../../../../../service/repomanager/project.repomanager.service';
import { VariableService } from '../../../../../service/variable/variable.service';
import { SharedModule } from '../../../../../shared/shared.module';
import { ToastService } from '../../../../../shared/toast/ToastService';
import { ProjectModule } from '../../../../project/project.module';
import { ApplicationModule } from '../../../application.module';
import { ApplicationRepositoryComponent } from './application.repo.component';

@Component({
    template: ''
})
class DummyComponent {
}


describe('CDS: Application Repo Component', () => {

    let injector: Injector;
    let toast: ToastService;

    beforeEach(() => {
        TestBed.configureTestingModule({
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
                PipelineService,
                EnvironmentService,
                VariableService,
                { provide: ToastService, useClass: MockToast },
                TranslateLoader,
                TranslateService,
                TranslateParser,
                RepoManagerService,
                Store,
                { provide: APP_BASE_HREF, useValue: '/' }
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
        });


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
            { 'name': 'repo1', 'fullname': 'frepo1' },
            { 'name': 'repo2', 'fullname': 'frepo2' },
            { 'name': 'repo3', 'fullname': 'frepo3' },
            { 'name': 'repo4', 'fullname': 'frepo4' },
            { 'name': 'repo5', 'fullname': 'frepo5' }
        ];

        let fixture = TestBed.createComponent(ApplicationRepositoryComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let app: Application = new Application();
        app.name = 'app';
        app.permission = 7;
        let p: Project = new Project();
        p.key = 'key1';
        p.name = 'proj1';

        let repoMan: RepositoriesManager = { name: 'RepoManager' };
        p.vcs_servers = new Array<RepositoriesManager>();
        p.vcs_servers.push(repoMan);

        fixture.componentInstance.application = app;
        fixture.componentInstance.project = p;

        fixture.componentInstance.ngOnInit();
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/key1/repositories_manager/RepoManager/repos';
        })).flush(mockResponse);

        expect(fixture.componentInstance.selectedRepoManager).toBe('RepoManager');
        expect(fixture.componentInstance.repos.length).toBe(5, 'Must have 5 repositories in list');

        // Select repo + link
        fixture.componentInstance.selectedRepo = 'frepo3';

        fixture.detectChanges();
        tick(50);

        spyOn(toast, 'success');
        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('button[name="addrepobtn"]').click();
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
        tick(100);
        expect(toast.success).toHaveBeenCalledTimes(2);
    }));
});

class MockRouter {
    public navigate() {
    }
}

class MockStore {
    constructor() {
    }

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
