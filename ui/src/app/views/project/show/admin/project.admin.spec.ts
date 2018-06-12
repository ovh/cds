/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {ToasterService} from 'angular2-toaster/angular2-toaster';
import {TranslateParser} from '@ngx-translate/core';
import {Observable} from 'rxjs/Observable';

import {ProjectAdminComponent} from './project.admin.component';
import {ProjectStore} from '../../../../service/project/project.store';
import {RepoManagerService} from '../../../../service/repomanager/project.repomanager.service';
import {ProjectService} from '../../../../service/project/project.service';
import {PipelineService} from '../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../service/environment/environment.service';
import {VariableService} from '../../../../service/variable/variable.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {ProjectModule} from '../../project.module';
import {SharedModule} from '../../../../shared/shared.module';
import {ServicesModule} from '../../../../service/services.module';
import {Project} from '../../../../model/project.model';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import 'rxjs/add/observable/of';
import {NavbarService} from '../../../../service/navbar/navbar.service';

describe('CDS: Project Admin Component', () => {

    let injector: Injector;
    let projectStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                NavbarService,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                ToasterService,
                TranslateService,
                TranslateParser,
                AuthentificationStore,
                { provide: ToastService, useClass: MockToastt}
            ],
            imports : [
                ProjectModule,
                SharedModule,
                ServicesModule,
                TranslateModule.forRoot(),
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


    it('it should update the project', () => {
        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectAdminComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let p: Project = new Project();
        p.key = 'key1';
        p.permission = 7;
        fixture.componentInstance.project = p;

        fixture.detectChanges(true);

        spyOn(projectStore, 'updateProject').and.callFake(() => {
            return Observable.of(p);
        });

        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('button[name="btnrename"]').click();

        expect(projectStore.updateProject).toHaveBeenCalledWith(p);
    });
});

class MockToastt {
    success(title: string, msg: string) {

    }
}
