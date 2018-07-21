import {CUSTOM_ELEMENTS_SCHEMA, NgModule, NO_ERRORS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowBreadCrumbComponent} from './breadcrumb/workflow.breadcrumb.component';
import {WorkflowGraphComponent} from './graph/workflow.graph.component';
import {WorkflowRunArtifactListComponent} from './run/node/artifact/artifact.list.component';
import {WorkflowNodeRunHistoryComponent} from './run/node/history/history.component';
import {WorkflowRunNodePipelineComponent} from './run/node/pipeline/node.pipeline.component';
import {WorkflowServiceLogComponent} from './run/node/pipeline/service/service.log.component';
import {WorkflowRunJobSpawnInfoComponent} from './run/node/pipeline/spawninfo/spawninfo.component';
import {WorkflowStepLogComponent} from './run/node/pipeline/step/step.log.component';
import {WorkflowRunJobVariableComponent} from './run/node/pipeline/variables/job.variables.component';
import {WorkflowNodeRunSummaryComponent} from './run/node/summary/run.summary.component';
import {WorkflowRunTestTableComponent} from './run/node/test/table/test.table.component';
import {WorkflowRunTestsResultComponent} from './run/node/test/tests.component';
import {WorkflowNodeRunComponent} from './run/node/workflow.run.node.component';
import {WorkflowRunSummaryComponent} from './run/summary/workflow.run.summary.component';
import {WorkflowRunComponent} from './run/workflow.run.component';
import {WorkflowAdminComponent} from './show/admin/workflow.admin.component';
import {WorkflowNotificationFormComponent} from './show/notification/form/workflow.notification.form.component';
import {WorkflowNotificationListComponent} from './show/notification/list/workflow.notification.list.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowSidebarCodeComponent} from './sidebar/code/sidebar.code.component';
import {WorkflowSidebarHookComponent} from './sidebar/edit/hook/workflow.sidebar.hook.component';
import {WorkflowSidebarEditJoinComponent} from './sidebar/edit/join/workflow.sidebar.edit.join.component';
import {WorkflowSidebarEditNodeComponent} from './sidebar/edit/node/workflow.sidebar.edit.node.component';
import {WorkflowSidebarEditComponent} from './sidebar/edit/workflow.sidebar.edit.component';
import {WorkflowSidebarRunHookComponent} from './sidebar/run/hook/workflow.sidebar.run.hook.component';
import {WorkflowSidebarRunListComponent} from './sidebar/run/list/workflow.sidebar.run.component';
import {ActionStepSummaryComponent} from './sidebar/run/node/stage/job/action/action.summary.component';
import {JobStepSummaryComponent} from './sidebar/run/node/stage/job/job.summary.component';
import {StageStepSummaryComponent} from './sidebar/run/node/stage/stage.summary.component';
import {WorkflowSidebarRunNodeComponent} from './sidebar/run/node/workflow.sidebar.run.node.component';
import {WorkflowComponent} from './workflow.component';
import {workflowRouting} from './workflow.routing';

@NgModule({
    declarations: [
        WorkflowAdminComponent,
        WorkflowComponent,
        WorkflowAddComponent,
        WorkflowBreadCrumbComponent,
        WorkflowGraphComponent,
        WorkflowRunComponent,
        WorkflowNodeRunComponent,
        WorkflowRunJobVariableComponent,
        WorkflowRunJobSpawnInfoComponent,
        WorkflowRunNodePipelineComponent,
        WorkflowRunArtifactListComponent,
        WorkflowRunTestsResultComponent,
        WorkflowRunTestTableComponent,
        WorkflowRunSummaryComponent,
        WorkflowSidebarEditComponent,
        WorkflowSidebarEditNodeComponent,
        WorkflowSidebarEditJoinComponent,
        WorkflowSidebarHookComponent,
        WorkflowSidebarRunHookComponent,
        WorkflowNodeRunHistoryComponent,
        WorkflowSidebarRunListComponent,
        WorkflowSidebarRunNodeComponent,
        WorkflowSidebarCodeComponent,
        StageStepSummaryComponent,
        JobStepSummaryComponent,
        ActionStepSummaryComponent,
        WorkflowShowComponent,
        WorkflowStepLogComponent,
        WorkflowServiceLogComponent,
        WorkflowNodeRunSummaryComponent,
        WorkflowNotificationFormComponent,
        WorkflowNotificationListComponent
    ],
    imports: [
        SharedModule,
        workflowRouting
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA,
        NO_ERRORS_SCHEMA
    ]
})
export class WorkflowModule {
}
