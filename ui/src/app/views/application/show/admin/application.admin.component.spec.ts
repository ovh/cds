/* tslint:disable:no-unused-variable */

import {RouterTestingModule} from '@angular/router/testing';
import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Router, ActivatedRoute} from '@angular/router';
import {Observable} from 'rxjs';
import {Injector, Component} from '@angular/core';
import {Application} from '../../../../model/application.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {ApplicationAdminComponent} from './application.admin.component';
import {ApplicationService} from '../../../../service/application/application.service';
import {SharedModule} from '../../../../shared/shared.module';
import {TranslateService, TranslateLoader} from 'ng2-translate/ng2-translate';
import {ToastService} from '../../../../shared/toast/ToastService';
import {Project} from '../../../../model/project.model';
import {RepoManagerService} from '../../../../service/repomanager/project.repomanager.service';
import {ApplicationModule} from '../../application.module';
import {Pipeline} from '../../../../model/pipeline.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {ProjectService} from '../../../../service/project/project.service';
import {TranslateParser} from 'ng2-translate';
import {ProjectModule} from '../../../project/project.module';
import {RepositoriesManager} from '../../../../model/repositories.model';

@Component({
    template: ''
})
class DummyComponent {
}

describe('CDS: Application Admin Component', () => {

    let injector: Injector;
    let appStore: ApplicationStore;
    let backend: MockBackend;
    let router: Router;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
                DummyComponent
            ],
            providers: [
                { provide: XHRBackend, useClass: MockBackend },
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                { provide: ToastService, useClass: MockToast },
                TranslateLoader,
                TranslateService,
                TranslateParser,
                RepoManagerService
            ],
            imports : [
                RouterTestingModule.withRoutes([
                    { path: 'project/:key', component: DummyComponent },
                    { path: 'project/:key/application/:appName', component: DummyComponent }
                ]),
                ApplicationModule,
                SharedModule
            ]
        });


        injector = getTestBed();
        backend = injector.get(XHRBackend);
        appStore = injector.get(ApplicationStore);
        router = injector.get(Router);
    });

    afterEach(() => {
        injector = undefined;
        appStore = undefined;
        backend = undefined;
        router = undefined;
    });

    it('Load component + renamed app', fakeAsync( () => {

            // Mock Http login request
            backend.connections.subscribe(connection => {
                connection.mockRespond(new Response(new ResponseOptions({ body : '{ "name": "appRenamed", "permission": 7 }'})));
            });

            let fixture = TestBed.createComponent(ApplicationAdminComponent);
            let component = fixture.debugElement.componentInstance;
            expect(component).toBeTruthy();

            let app: Application = new Application();
            app.name = 'app';
            app.permission = 7;
            let p: Project = new Project();
            p.key = 'key1';
            p.name = 'proj1';
            p.repositories_manager = new Array<RepositoriesManager>();
            let rm = new RepositoriesManager();
            p.repositories_manager.push(rm);

            let pip: Pipeline = new Pipeline();
            pip.name = 'myPipeline';
            p.pipelines = new Array<Pipeline>();
            p.pipelines.push(pip);

            fixture.componentInstance.application = app;
            fixture.componentInstance.project = p;

            fixture.detectChanges();
            tick(50);

            let compiled = fixture.debugElement.nativeElement;

            let inputName = compiled.querySelector('input[name="formApplicationUpdateName"]');
            inputName.value = 'appRenamed';
            inputName.dispatchEvent(new Event('input'));

            spyOn(router, 'navigate');
            compiled.querySelector('button[name="updateNameButton"]').click();

            expect(router.navigate).toHaveBeenCalledWith(['/project', 'key1', 'application', 'appRenamed']);

            tick(50);

    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}
