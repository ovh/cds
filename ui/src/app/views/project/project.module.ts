import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { DragDropModule } from '@angular/cdk/drag-drop';
import { SharedModule } from 'app/shared/shared.module';
import { ProjectAddComponent } from './add/project.add.component';
import { projectRouting } from './project.routing';
import { ProjectRepoManagerListComponent } from './settings/advanced/repomanager/list/project.repomanager.list.component';
import { ProjectApplicationListComponent } from './show/application/application.list.component';
import { ProjectEnvironmentListComponent } from './show/environment/environment.list.component';
import { ProjectIntegrationsComponent } from './settings/integrations/project.integrations.component';
import { ProjectKeysComponent } from './settings/keys/project.keys.component';
import { ProjectPermissionsComponent } from './show/permission/permission.component';
import { ProjectPipelinesComponent } from './show/pipeline/pipeline.list.component';
import { ProjectShowComponent } from './show/project.component';
import { ProjectVariablesComponent } from './show/variable/variable.list.component';
import { ProjectWorkflowListBlocsComponent } from './show/workflow/blocs/workflow.list.blocs.component';
import { ProjectWorkflowListLabelsComponent } from './show/workflow/labels/workflow.list.labels.component';
import { ProjectWorkflowListLinesComponent } from './show/workflow/lines/workflow.list.lines.component';
import { ProjectWorkflowListComponent } from './show/workflow/workflow.list.component';
import { ProjectComponent } from './project.component';
import { ProjectActivityBarComponent } from './activity-bar/activity-bar.component';
import { ProjectSettingsComponent } from './settings/settings.component';
import { ProjectVariableSetsComponent } from './settings/variablesets/variablesets.component';
import { ProjectVariableSetItemsComponent } from './settings/variablesets/items/variableset.item.component';
import { ProjectAdvancedComponent } from './settings/advanced/project.advanced.component';
import { ProjectRepoManagerFormComponent } from './settings/advanced/repomanager/from/project.repomanager.form.component';
import { ProjectExistsGuard, ProjectGuard, ProjectV2Guard } from './project.guard';
import { ProjectConcurrenciesComponent } from './settings/concurrency/concurrencies.components';
import { ProjectConcurrencyFormComponent } from './settings/concurrency/concurrency.form.component';
import { ProjectWebhooksComponent } from './settings/webhooks/webhooks.component';
import { ProjectRunRetentionComponent } from './settings/retention/retention.component';
import { ProjectRunRetentionReportComponent } from './settings/retention/retention.report.component';
import { ProjectV2ExploreComponent } from '../projectv2/explore/explore.component';
import { ProjectV2ExploreEntityComponent } from '../projectv2/explore/explore-entity.component';
import { ProjectV2ExploreOverviewComponent } from '../projectv2/explore/explore-overview.component';
import { ProjectV2ExploreRepositoryComponent } from '../projectv2/explore/explore-repository.component';
import { ProjectV2ExploreSidebarComponent } from '../projectv2/explore/explore-sidebar.component';
import { ProjectV2RepositoryAddComponent } from '../projectv2/explore/repository-add/repository-add.component';
import { ProjectV2RunComponent } from '../projectv2/run/run.component';
import { ProjectV2RunListComponent } from '../projectv2/run-list/run-list.component';
import { ProjectV2RunListSidebarComponent } from '../projectv2/run-list/run-list-sidebar.component';
import { ProjectV2RunStartComponent } from '../projectv2/run-start/run-start.component';
import { RunGateInputsComponent } from '../projectv2/run-start/run-gate-inputs.component';
import { ProjectV2TriggerAnalysisComponent } from '../projectv2/explore/trigger-analysis/trigger-analysis.component';
import { RunContextsComponent } from '../projectv2/run/run-contexts.component';
import { RunGateComponent } from '../projectv2/run/run-gate.component';
import { RunHookComponent } from '../projectv2/run/run-hook.component';
import { RunInfoComponent } from '../projectv2/run/run-info.component';
import { RunJobComponent } from '../projectv2/run/run-job.component';
import { RunResultComponent } from '../projectv2/run/run-result.component';
import { RunResultsComponent } from '../projectv2/run/run-results.component';
import { RunSourcesComponent } from '../projectv2/run/run-sources.component';
import { RunTestComponent } from '../projectv2/run/run-test.component';
import { RunTestsComponent } from '../projectv2/run/run-tests.component';
import { GraphModule } from '../../../../libs/workflow-graph/src/public-api';

@NgModule({
    declarations: [
        ProjectActivityBarComponent,
        ProjectAddComponent,
        ProjectAdvancedComponent,
        ProjectApplicationListComponent,
        ProjectComponent,
        ProjectConcurrenciesComponent,
        ProjectConcurrencyFormComponent,
        ProjectEnvironmentListComponent,
        ProjectEnvironmentListComponent,
        ProjectIntegrationsComponent,
        ProjectKeysComponent,
        ProjectPermissionsComponent,
        ProjectPipelinesComponent,
        ProjectRepoManagerFormComponent,
        ProjectRepoManagerListComponent,
        ProjectRunRetentionComponent,
        ProjectRunRetentionReportComponent,
        ProjectSettingsComponent,
        ProjectShowComponent,
        ProjectVariablesComponent,
        ProjectVariableSetItemsComponent,
        ProjectVariableSetsComponent,
        ProjectWebhooksComponent,
        ProjectWorkflowListBlocsComponent,
        ProjectWorkflowListComponent,
        ProjectWorkflowListLabelsComponent,
        ProjectWorkflowListLinesComponent,
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
        RunGateInputsComponent,
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
        SharedModule,
        RouterModule,
        DragDropModule,
        GraphModule,
        projectRouting
    ],
    providers: [
        ProjectExistsGuard,
        ProjectGuard,
        ProjectV2Guard
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectModule { }
