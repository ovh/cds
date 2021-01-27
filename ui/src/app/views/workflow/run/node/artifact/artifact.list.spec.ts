/* eslint-disable @typescript-eslint/no-unused-vars */
import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { TestBed, waitForAsync } from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateLoader, TranslateModule, TranslateParser, TranslateService} from '@ngx-translate/core';
import { NgxsModule, Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { ApplicationService } from 'app/service/application/application.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { RouterService } from 'app/service/router/router.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import {SharedModule} from 'app/shared/shared.module';
import { ApplicationsState } from 'app/store/applications.state';
import { EnvironmentState } from 'app/store/environment.state';
import { PipelinesState } from 'app/store/pipelines.state';
import * as ProjectAction from 'app/store/project.action';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState } from 'app/store/workflow.state';
import {WorkflowModule} from 'app/views/workflow/workflow.module';
import {WorkflowRunArtifactListComponent} from './artifact.list.component';

describe('CDS: Artifact List', () => {

    let store: Store;
    let http: HttpTestingController;

    beforeEach(waitForAsync( () => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
                ProjectService, PipelineService, EnvironmentService, ApplicationService, EnvironmentService,
                NavbarService, WorkflowService, WorkflowRunService, ProjectStore, RouterService
            ],
            imports: [
                HttpClientTestingModule,
                WorkflowModule,
                NgxsModule.forRoot([ProjectState, ApplicationsState, PipelinesState, WorkflowState, EnvironmentState]),
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        }).compileComponents();
        store = TestBed.inject(Store);
        http = TestBed.inject(HttpTestingController);
    }));

    it('should load component', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));

        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });
        console.log(store.selectSnapshot(ProjectState.projectSnapshot));

        // Create component
        let fixture = TestBed.createComponent(WorkflowRunArtifactListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();
    }));
});
