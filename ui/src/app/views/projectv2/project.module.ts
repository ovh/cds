import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from 'app/shared/shared.module';
import { ProjectV2RunComponent } from "./run/run.component";
import { ProjectV2ExploreSidebarComponent } from './explore/explore-sidebar.component';
import { ProjectV2ExploreEntityComponent } from './explore/explore-entity.component';
import { RunJobComponent } from "./run/run-job.component";
import { RunJobLogsComponent } from "./run/run-job-logs.component";
import { RunGateComponent } from "./run/gate/gate.component";
import { ProjectV2RunListComponent } from './run-list/run-list.component';
import { WorkflowGraphModule } from '../../../../libs/workflow-graph/src/public-api';
import { ProjectV2ExploreComponent } from './explore/explore.component';
import { ProjectV2RunListSidebarComponent } from './run-list/run-list-sidebar.component';
import { RunHookComponent } from './run/run-hook.component';
import { RunResultComponent } from './run/run-result.component';
import { RunWorkflowComponent } from './run/run-workflow.component';
import { RunContextsComponent } from './run/run-contexts.component';
import { RunTestsComponent } from './run/run-tests.component';
import { RunTestComponent } from './run/run-test.component';
import { ProjectV2ExploreRepositoryComponent } from './explore/explore-repository.component';
import { ProjectV2ExploreRepositoryAddComponent } from './explore/explore-repository-add.component';
import { ProjectV2ExploreEntityWorkflowComponent } from './explore/explore-entity-workflow.component';
import { EntityFormComponent } from './explore/entity/entity-form.component';
import { EntityJSONFormComponent } from './explore/entity/entity-json-form.component';
import { EntityJSONFormFieldComponent } from './explore/entity/entity-json-form-field.component';
import { ProjectV2RunStartComponent } from './run-start/run-start.component';
import { ProjectV2TriggerAnalysisComponent } from './explore/trigger-analysis/trigger-analysis.component';

@NgModule({
    declarations: [
        EntityFormComponent,
        EntityJSONFormComponent,
        EntityJSONFormFieldComponent,
        ProjectV2ExploreComponent,
        ProjectV2ExploreEntityComponent,
        ProjectV2ExploreEntityWorkflowComponent,
        ProjectV2ExploreRepositoryAddComponent,
        ProjectV2ExploreRepositoryComponent,
        ProjectV2ExploreSidebarComponent,
        ProjectV2RunComponent,
        ProjectV2RunListComponent,
        ProjectV2RunListSidebarComponent,
        ProjectV2RunStartComponent,
        ProjectV2TriggerAnalysisComponent,
        RunContextsComponent,
        RunGateComponent,
        RunHookComponent,
        RunJobComponent,
        RunJobLogsComponent,
        RunResultComponent,
        RunTestComponent,
        RunTestsComponent,
        RunWorkflowComponent
    ],
    imports: [
        RouterModule,
        SharedModule,
        WorkflowGraphModule
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectV2Module { }
