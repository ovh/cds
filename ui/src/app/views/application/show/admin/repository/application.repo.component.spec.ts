/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, getTestBed, tick, inject} from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector, NO_ERRORS_SCHEMA, Component} from '@angular/core';
import {TranslateService, TranslateLoader} from 'ng2-translate/ng2-translate';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ApplicationRepositoryComponent} from './application.repo.component';
import {ApplicationService} from '../../../../../service/application/application.service';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {RepoManagerService} from '../../../../../service/repomanager/project.repomanager.service';
import {SharedModule} from '../../../../../shared/shared.module';
import {Application} from '../../../../../model/application.model';
import {Project} from '../../../../../model/project.model';
import {RepositoriesManager} from '../../../../../model/repositories.model';
import {Observable} from 'rxjs/Rx';
import {ApplicationModule} from '../../../application.module';
import {TranslateParser} from 'ng2-translate';
import {ProjectModule} from '../../../../project/project.module';

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
                { provide: XHRBackend, useClass: MockBackend },
                { provide: ApplicationStore, useClass: MockStore },
                ApplicationService,
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
                SharedModule
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

    it('Load component + select repository', fakeAsync(inject([XHRBackend], (backend: MockBackend) => {
        TestBed.compileComponents().then(() => {
            let call = 0;
            // Mock Http login request
            backend.connections.subscribe(connection => {
                call++;
                connection.mockRespond(new Response(new ResponseOptions({
                    body: `[
                    { "name" : "repo1", "fullname": "frepo1" },
                    { "name" : "repo2", "fullname": "frepo2" },
                    { "name" : "repo3", "fullname": "frepo3" },
                    { "name" : "repo4", "fullname": "frepo4" },
                    { "name" : "repo5", "fullname": "frepo5" }
                ]`
                })));
            });

            let fixture = TestBed.createComponent(ApplicationRepositoryComponent);
            let component = fixture.debugElement.componentInstance;
            expect(component).toBeTruthy();


            let app: Application = new Application();
            app.name = 'app';
            app.permission = 7;
            let p: Project = new Project();
            p.key = 'key1';
            p.name = 'proj1';

            let repoMan: RepositoriesManager = {id: 1, name: 'RepoManager', type: 'typeR', url: 'foo.bar'};
            p.repositories_manager = new Array<RepositoriesManager>();
            p.repositories_manager.push(repoMan);

            fixture.componentInstance.application = app;
            fixture.componentInstance.project = p;

            // Init component
            expect(call).toBe(0, 'No http call yet');
            fixture.componentInstance.ngOnInit();
            expect(call).toBe(1, 'Get repo list must have been called');
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

            fixture.componentInstance.application.repositories_manager = repoMan;
            fixture.componentInstance.application.repository_fullname = 'frepo3';

            tick(100);

            // Detach repo
            compiled.querySelector('.ui.red.button').click();
            fixture.detectChanges();
            tick(50);
            compiled.querySelector('.ui.red.button.active').click();
            expect(toast.success).toHaveBeenCalledTimes(2);
        });
    })));
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
