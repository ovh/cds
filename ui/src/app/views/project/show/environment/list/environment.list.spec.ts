/* tslint:disable:no-unused-variable */

import { HttpClientTestingModule } from '@angular/common/http/testing';
import { getTestBed, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { ToasterService } from 'angular2-toaster';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import { Environment } from '../../../../../model/environment.model';
import { Project } from '../../../../../model/project.model';
import { EnvironmentService } from '../../../../../service/environment/environment.service';
import { NavbarService } from '../../../../../service/navbar/navbar.service';
import { PipelineService } from '../../../../../service/pipeline/pipeline.service';
import { ProjectService } from '../../../../../service/project/project.service';
import { ProjectStore } from '../../../../../service/project/project.store';
import { ServicesModule, WorkflowRunService } from '../../../../../service/services.module';
import { VariableService } from '../../../../../service/variable/variable.service';
import { SharedModule } from '../../../../../shared/shared.module';
import { ToastService } from '../../../../../shared/toast/ToastService';
import { ProjectModule } from '../../../project.module';
import { ProjectEnvironmentListComponent } from './environment.list.component';
import {WorkflowService} from 'app/service/workflow/workflow.service';

describe('CDS: Environment List Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ProjectStore,
                ProjectService,
                TranslateService,
                ToasterService,
                ToastService,
                TranslateLoader,
                TranslateParser,
                VariableService,
                NavbarService,
                PipelineService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                { provide: Router, useClass: MockRouter },
                EnvironmentService,
                WorkflowService,
                WorkflowRunService
            ],
            imports: [
                ProjectModule,
                SharedModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                ServicesModule,
                RouterTestingModule.withRoutes([
                    { path: 'project/:key', component: ProjectEnvironmentListComponent },
                ]),
                HttpClientTestingModule
            ]
        });

        this.injector = getTestBed();
    });

    afterEach(() => {
        this.injector = undefined;
    });

    it('should load component', () => {
        // Create component
        let fixture = TestBed.createComponent(ProjectEnvironmentListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();


        let project = new Project();
        project.key = 'key1';

        let envs = new Array<Environment>();
        let e = new Environment();
        e.name = 'prod';
        envs.push(e);
        project.environments = envs;

        fixture.componentInstance.project = project;
        fixture.componentInstance.ngOnInit();
    });
});

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();

        this.queryParams = of({ envName: 'prod' });
    }
}

class MockRouter {
    public navigate() {
    }
}
