/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed, tick, inject} from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import {Injector, NO_ERRORS_SCHEMA, Component} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateModule} from '@ngx-translate/core';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ApplicationRepositoryComponent} from './application.repo.component';
import {ApplicationService} from '../../../../../service/application/application.service';
import {PipelineService} from '../../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../../service/environment/environment.service';
import {VariableService} from '../../../../../service/variable/variable.service';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {RepoManagerService} from '../../../../../service/repomanager/project.repomanager.service';
import {SharedModule} from '../../../../../shared/shared.module';
import {Application} from '../../../../../model/application.model';
import {Project} from '../../../../../model/project.model';
import {RepositoriesManager} from '../../../../../model/repositories.model';
import {Observable} from 'rxjs/Observable';
import {ApplicationModule} from '../../../application.module';
import {TranslateParser} from '@ngx-translate/core';
import {ProjectModule} from '../../../../project/project.module';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import 'rxjs/add/observable/of';

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
                PipelineService,
                EnvironmentService,
                VariableService,
                { provide: ToastService, useClass: MockToast },
                TranslateLoader,
                TranslateService,
                TranslateParser,
                RepoManagerService
            ],
            imports : [
                RouterTestingModule.withRoutes([
                    { path: 'project/:key', component: DummyComponent }
                ]),
                ProjectModule,
                ApplicationModule,
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ],
            schemas: [ NO_ERRORS_SCHEMA ]
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
            { 'name' : 'repo1', 'fullname': 'frepo1' },
            { 'name' : 'repo2', 'fullname': 'frepo2' },
            { 'name' : 'repo3', 'fullname': 'frepo3' },
            { 'name' : 'repo4', 'fullname': 'frepo4' },
            { 'name' : 'repo5', 'fullname': 'frepo5' }
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

        let repoMan: RepositoriesManager = {name: 'RepoManager'};
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

        expect(toast.success).toHaveBeenCalledTimes(1);

        fixture.componentInstance.application.vcs_server = repoMan.name;
        fixture.componentInstance.application.repository_fullname = 'frepo3';

        tick(100);

        // Detach repo
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges();
        tick(50);
        compiled.querySelector('.ui.red.button.active').click();
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
        return  Observable.of({ name: 'app'});
    }
    removeRepository(key: string, currentName: string, repoManName: string, repoFullname: string) {
        return  Observable.of({ name: 'app'});
    }
}

class MockToast {
    success(title: string, msg: string) {

    }
}
