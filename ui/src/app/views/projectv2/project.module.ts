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
import { RunJobComponent } from "./run/run-job.component";
import { RunJobLogsComponent } from "./run/run-job-logs.component";
import { RunGateComponent } from "./run/gate/gate.component";
import { ProjectV2ActivityBarComponent } from './activity-bar/activity-bar.component';
import { ProjectV2WorkflowRunListComponent } from './run-list/run-list.component';
import { WorkflowGraphModule } from '../../../../libs/workflow-graph/src/public-api';
import { ProjectV2ExploreComponent } from './explore/explore.component';
import { ProjectV2WorkflowRunListSidebarComponent } from './run-list/run-list-sidebar.component';
import { RunHookComponent } from './run/run-hook.component';

@NgModule({
    declarations: [
        ProjectV2ActionShowComponent,
        ProjectV2ActivityBarComponent,
        ProjectV2ExploreComponent,
        ProjectV2RepositoryAddComponent,
        ProjectV2RepositoryShowComponent,
        ProjectV2ShowComponent,
        ProjectV2SidebarComponent,
        ProjectV2TopMenuComponent,
        ProjectV2WorkerModelShowComponent,
        ProjectV2WorkflowRunComponent,
        ProjectV2WorkflowRunListComponent,
        ProjectV2WorkflowRunListSidebarComponent,
        ProjectV2WorkflowShowComponent,
        ProjectWorkflowEntityComponent,
        RunGateComponent,
        RunHookComponent,
        RunJobComponent,
        RunJobLogsComponent
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
