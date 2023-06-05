import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from 'app/shared/shared.module';
import { projectV2Routing } from './project.routing';
import { ProjectV2ShowComponent } from 'app/views/projectv2/project.component';
import { ProjectV2TopMenuComponent } from 'app/views/projectv2/top-menu/project.top.menu.component';
import { ProjectV2SidebarComponent } from 'app/views/projectv2/sidebar/sidebar.component';
import {
    ProjectV2RepositoryAddComponent
} from 'app/views/projectv2/vcs/repository/project.repository.add.component';
import {
    ProjectV2RepositoryShowComponent
} from 'app/views/projectv2/vcs/repository/show/project.repository.show.component';
import { ProjectV2WorkerModelShowComponent } from "./vcs/repository/workermodel/show/project.workermodel.show.component";
import {ProjectV2ActionShowComponent} from "./vcs/repository/action/show/project.action.show.component";
import {ProjectV2WorkflowShowComponent} from "./vcs/repository/workflow/show/project.workflow.show.component";
import {
    ProjectV2WorkflowForkJoinNodeComponent,
} from "./vcs/repository/workflow/show/graph/node/fork-join-node.components";
import {
    ProjectV2WorkflowJobNodeComponent
} from "./vcs/repository/workflow/show/graph/node/job-node.component";
import {
    ProjectV2WorkflowStagesGraphComponent
} from "./vcs/repository/workflow/show/graph/stages-graph.component";
import {
    ProjectV2WorkflowJobsGraphComponent
} from "./vcs/repository/workflow/show/graph/jobs-graph.component";

@NgModule({
    declarations: [
        ProjectV2RepositoryAddComponent,
        ProjectV2RepositoryShowComponent,
        ProjectV2ShowComponent,
        ProjectV2SidebarComponent,
        ProjectV2TopMenuComponent,
        ProjectV2WorkerModelShowComponent,
        ProjectV2ActionShowComponent,
        ProjectV2WorkflowShowComponent,

        ProjectV2WorkflowForkJoinNodeComponent,
        ProjectV2WorkflowJobNodeComponent,
        ProjectV2WorkflowStagesGraphComponent,
        ProjectV2WorkflowJobsGraphComponent
    ],
    imports: [
        SharedModule,
        RouterModule,
        projectV2Routing,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectV2Module { }
