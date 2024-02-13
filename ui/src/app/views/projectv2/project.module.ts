import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from 'app/shared/shared.module';
import { projectV2Routing } from './project.routing';
import { ProjectV2ShowComponent } from 'app/views/projectv2/project.component';
import { ProjectV2TopMenuComponent } from 'app/views/projectv2/top-menu/project.top.menu.component';
import {
    ProjectV2RepositoryAddComponent
} from 'app/views/projectv2/vcs/repository/project.repository.add.component';
import {
    ProjectV2RepositoryShowComponent
} from 'app/views/projectv2/vcs/repository/show/project.repository.show.component';
import { ProjectV2WorkerModelShowComponent } from "./vcs/repository/workermodel/show/project.workermodel.show.component";
import { ProjectV2ActionShowComponent } from "./vcs/repository/action/show/project.action.show.component";
import { ProjectV2WorkflowShowComponent } from "./vcs/repository/workflow/show/project.workflow.show.component";
import { ProjectWorkflowEntityComponent } from "./vcs/repository/workflow/show/entity/project.workflow.entity.component";
import { ProjectV2WorkflowRunComponent } from "./run/project.run.component";
import { ProjectV2SidebarComponent } from './sidebar/workspace/sidebar.component';
import { ProjectV2SidebarRunComponent } from "./sidebar/run/sidebar.run.component";
import { RunJobComponent } from "./run/run-job.component";
import { RunJobLogsComponent } from "./run/run-job-logs.component";
import { RunGateComponent } from "./run/gate/gate.component";
import { ProjectV2LeftMenuComponent } from './left-menu/left-menu.component';
import { ProjectV2WorkflowRunListComponent } from './run-list/run-list.component';
import { WorkflowGraphModule } from '../../../../libs/workflow-graph/src/public-api';

@NgModule({
    declarations: [
        ProjectV2ActionShowComponent,
        ProjectV2LeftMenuComponent,
        ProjectV2RepositoryAddComponent,
        ProjectV2RepositoryShowComponent,
        ProjectV2ShowComponent,
        ProjectV2SidebarComponent,
        ProjectV2SidebarRunComponent,
        ProjectV2TopMenuComponent,
        ProjectV2WorkerModelShowComponent,
        ProjectV2WorkflowRunComponent,
        ProjectV2WorkflowShowComponent,
        ProjectWorkflowEntityComponent,
        RunGateComponent,
        RunJobComponent,
        RunJobLogsComponent,
        ProjectV2WorkflowRunListComponent
    ],
    imports: [
        SharedModule,
        RouterModule,
        projectV2Routing,
        WorkflowGraphModule
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectV2Module { }
