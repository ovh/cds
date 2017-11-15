/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {ProjectStore} from '../../../../../service/project/project.store';
import {ProjectService} from '../../../../../service/project/project.service';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {ProjectModule} from '../../../project.module';
import {ProjectEnvironmentFormComponent} from './environment.form.component';
import {Project} from '../../../../../model/project.model';
import {Observable} from 'rxjs/Observable';
import {SharedModule} from '../../../../../shared/shared.module';
import {Environment} from '../../../../../model/environment.model';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Environment From Component', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ProjectStore,
                ProjectService,
                { provide: ToastService, useClass: MockToast },
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                ProjectModule,
                SharedModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

        this.injector = getTestBed();
    });

    afterEach(() => {
        this.injector = undefined;
    });

    it('Create new environment', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ProjectEnvironmentFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let project = new Project();
        project.key = 'key1';
        fixture.componentInstance.project = project;

        let env = new Environment();
        env.name = 'Production';
        fixture.componentInstance.newEnvironment = env;

        let projStore: ProjectStore = this.injector.get(ProjectStore);
        spyOn(projStore, 'addProjectEnvironment').and.callFake(() => {
           let p = new Project();
           return Observable.of(p);
        });

        fixture.debugElement.nativeElement.querySelector('.ui.green.button').click();

        expect(projStore.addProjectEnvironment).toHaveBeenCalledWith('key1', env);
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

