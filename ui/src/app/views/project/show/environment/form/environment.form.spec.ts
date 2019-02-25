/* tslint:disable:no-unused-variable */

import { HttpClientTestingModule } from '@angular/common/http/testing';
import { Injector } from '@angular/core';
import { getTestBed, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddEnvironmentInProject } from 'app/store/project.action';
import { NgxsStoreModule } from 'app/store/store.module';
import 'rxjs/add/observable/of';
import { Observable } from 'rxjs/Observable';
import { Environment } from '../../../../../model/environment.model';
import { Project } from '../../../../../model/project.model';
import { EnvironmentService } from '../../../../../service/environment/environment.service';
import { NavbarService } from '../../../../../service/navbar/navbar.service';
import { PipelineService } from '../../../../../service/pipeline/pipeline.service';
import { ProjectService } from '../../../../../service/project/project.service';
import { ProjectStore } from '../../../../../service/project/project.store';
import { ServicesModule } from '../../../../../service/services.module';
import { VariableService } from '../../../../../service/variable/variable.service';
import { SharedModule } from '../../../../../shared/shared.module';
import { ToastService } from '../../../../../shared/toast/ToastService';
import { ProjectModule } from '../../../project.module';
import { ProjectEnvironmentFormComponent } from './environment.form.component';

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
                TranslateLoader,
                TranslateParser,
                VariableService,
                PipelineService,
                NavbarService,
                EnvironmentService
            ],
            imports: [
                ProjectModule,
                SharedModule,
                NgxsStoreModule,
                ServicesModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

        this.injector = getTestBed();
    });

    afterEach(() => {
        this.injector = undefined;
    });

    it('Create new environment', () => {
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

        let store: Store = this.injector.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => {
            return Observable.of(null);
        });

        fixture.debugElement.nativeElement.querySelector('.ui.green.button').click();

        expect(store.dispatch).toHaveBeenCalledWith(
            new AddEnvironmentInProject({
                projectKey: 'key1',
                environment: env
            })
        );
    });
});

class MockToast {
    success(title: string, msg: string) {

    }
}
