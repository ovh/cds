/* tslint:disable:no-unused-variable */

import {RouterTestingModule} from '@angular/router/testing';
import {getTestBed, TestBed} from '@angular/core/testing';
import {Router} from '@angular/router';
import {Component, Injector} from '@angular/core';
import {Application} from '../../../../model/application.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {ApplicationAdminComponent} from './application.admin.component';
import {ApplicationService} from '../../../../service/application/application.service';
import {SharedModule} from '../../../../shared/shared.module';
import {TranslateLoader, TranslateModule, TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../../shared/toast/ToastService';
import {Project} from '../../../../model/project.model';
import {RepoManagerService} from '../../../../service/repomanager/project.repomanager.service';
import {ApplicationModule} from '../../application.module';
import {ServicesModule} from '../../../../service/services.module';
import {Pipeline} from '../../../../model/pipeline.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {ProjectService} from '../../../../service/project/project.service';
import {PipelineService} from '../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../service/environment/environment.service';
import {VariableService} from '../../../../service/variable/variable.service';
import {TranslateParser} from '@ngx-translate/core';
import {RepositoriesManager} from '../../../../model/repositories.model';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import {ApplicationMigrateService} from '../../../../service/application/application.migration.service';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {NavbarService} from '../../../../service/navbar/navbar.service';

@Component({
    template: ''
})
class DummyComponent {
}

describe('CDS: Application Admin Component', () => {

    let injector: Injector;
    let appStore: ApplicationStore;
    let router: Router;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
                DummyComponent
            ],
            providers: [
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                PipelineService,
                EnvironmentService,
                NavbarService,
                VariableService,
                {provide: ToastService, useClass: MockToast},
                TranslateLoader,
                TranslateService,
                TranslateParser,
                RepoManagerService,
                ApplicationMigrateService,
                AuthentificationStore
            ],
            imports: [
                RouterTestingModule.withRoutes([
                    {path: 'project/:key', component: DummyComponent},
                    {path: 'project/:key/application/:appName', component: DummyComponent}
                ]),
                ApplicationModule,
                ServicesModule,
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        });


        injector = getTestBed();
        appStore = injector.get(ApplicationStore);
        router = injector.get(Router);
    });

    afterEach(() => {
        injector = undefined;
        appStore = undefined;
        router = undefined;
    });

    it('Load component + renamed app', () => {
        const http = TestBed.get(HttpTestingController);

        let appRenamed = new Application();
        appRenamed.name = 'appRenamed';
        appRenamed.permission = 7;


        let fixture = TestBed.createComponent(ApplicationAdminComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let app: Application = new Application();
        app.name = 'app';
        app.permission = 7;
        let p: Project = new Project();
        p.key = 'key1';
        p.name = 'proj1';
        p.vcs_servers = new Array<RepositoriesManager>();
        let rm = new RepositoriesManager();
        p.vcs_servers.push(rm);

        let pip: Pipeline = new Pipeline();
        pip.name = 'myPipeline';
        p.pipelines = new Array<Pipeline>();
        p.pipelines.push(pip);

        fixture.componentInstance.application = app;
        fixture.componentInstance.project = p;
        fixture.componentInstance.newName = 'appRenamed';


        spyOn(router, 'navigate');

        let compiled = fixture.debugElement.nativeElement;

        compiled.querySelector('button[name="updateNameButton"]').click();

        http.expectOne('/project/key1/application/app').flush(appRenamed);

        expect(router.navigate).toHaveBeenCalledWith(['/project', 'key1', 'application', 'appRenamed']);
    });
});

class MockToast {
    success(title: string, msg: string) {

    }
}
