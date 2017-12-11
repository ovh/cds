/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import { TranslateService, TranslateLoader} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
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

describe('CDS: Project Admin Component', () => {

    let injector: Injector;
    let backend: MockBackend;
    let projectStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                ToasterService,
                TranslateService,
                TranslateParser,
                AuthentificationStore,
                { provide: ToastService, useClass: MockToast}
            ],
            imports : [
                ProjectModule,
                SharedModule,
                ServicesModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        projectStore = undefined;
    });


    it('it should update the project', fakeAsync( () => {
        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectAdminComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let p: Project = new Project();
        p.key = 'key1';
        p.permission = 7;
        fixture.componentInstance.project = p;

        fixture.detectChanges();
        tick(250);

        spyOn(projectStore, 'updateProject').and.callFake(() => {
            return Observable.of(p);
        });

        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('button[name="btnrename"]').click();

        expect(projectStore.updateProject).toHaveBeenCalledWith(p);
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}
