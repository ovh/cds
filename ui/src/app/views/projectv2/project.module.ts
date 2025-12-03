import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from 'app/shared/shared.module';
import { ProjectV2RunComponent } from "./run/run.component";
import { ProjectV2ExploreSidebarComponent } from './explore/explore-sidebar.component';
import { ProjectV2ExploreEntityComponent } from './explore/explore-entity.component';
import { RunJobComponent } from "./run/run-job.component";
import { ProjectV2RunListComponent } from './run-list/run-list.component';
import { ProjectV2ExploreComponent } from './explore/explore.component';
import { ProjectV2RunListSidebarComponent } from './run-list/run-list-sidebar.component';
import { RunHookComponent } from './run/run-hook.component';
import { RunResultComponent } from './run/run-result.component';
import { RunSourcesComponent } from './run/run-sources.component';
import { RunContextsComponent } from './run/run-contexts.component';
import { RunTestsComponent } from './run/run-tests.component';
import { RunTestComponent } from './run/run-test.component';
import { ProjectV2ExploreRepositoryComponent } from './explore/explore-repository.component';
import { ProjectV2RepositoryAddComponent } from './explore/repository-add/repository-add.component';
import { ProjectV2RunStartComponent } from './run-start/run-start.component';
import { ProjectV2TriggerAnalysisComponent } from './explore/trigger-analysis/trigger-analysis.component';
import { RunResultsComponent } from './run/run-results.component';
import { RunInfoComponent } from './run/run-info.component';
import { GraphModule } from '../../../../libs/workflow-graph/src/public-api';
import { RunGateComponent } from './run/run-gate.component';
import { ProjectV2ExploreOverviewComponent } from './explore/explore-overview.component';

@NgModule({
    declarations: [
        ProjectV2ExploreComponent,
        ProjectV2ExploreEntityComponent,
        ProjectV2ExploreOverviewComponent,
        ProjectV2ExploreRepositoryComponent,
        ProjectV2ExploreSidebarComponent,
        ProjectV2RepositoryAddComponent,
        ProjectV2RunComponent,
        ProjectV2RunListComponent,
        ProjectV2RunListSidebarComponent,
        ProjectV2RunStartComponent,
        ProjectV2TriggerAnalysisComponent,
        RunContextsComponent,
        RunGateComponent,
        RunHookComponent,
        RunInfoComponent,
        RunJobComponent,
        RunResultComponent,
        RunResultsComponent,
        RunSourcesComponent,
        RunTestComponent,
        RunTestsComponent
    ],
    imports: [
        RouterModule,
        SharedModule,
        GraphModule
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectV2Module { }
