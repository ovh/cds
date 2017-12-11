import {CUSTOM_ELEMENTS_SCHEMA, NgModule, NO_ERRORS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {workflowRouting} from './workflow.routing';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowGraphComponent} from './graph/workflow.graph.component';
import {WorkflowRunComponent} from './run/workflow.run.component';
import {WorkflowNodeRunComponent} from './run/node/workflow.run.node.component';
import {WorkflowRunNodePipelineComponent} from './run/node/pipeline/node.pipeline.component';
import {WorkflowBreadCrumbComponent} from './breadcrumb/workflow.breadcrumb.component';
import {WorkflowRunJobVariableComponent} from './run/node/pipeline/variables/job.variables.component';
import {WorkflowRunJobSpawnInfoComponent} from './run/node/pipeline/spawninfo/spawninfo.component';
import {WorkflowStepLogComponent} from './run/node/pipeline/step/step.log.component';
import {WorkflowComponent} from './workflow.compoment';
import {WorkflowSidebarRunListComponent} from './sidebar/run/list/workflow.sidebar.run.component';
import {WorkflowSidebarRunNodeComponent} from './sidebar/run/node/workflow.sidebar.run.node.component';
import {WorkflowSidebarEditComponent} from './sidebar/edit/workflow.sidebar.edit.component';
import {WorkflowSidebarEditJoinComponent} from './sidebar/edit/join/workflow.sidebar.edit.join.component';
import {WorkflowSidebarEditNodeComponent} from './sidebar/edit/node/workflow.sidebar.edit.node.component';
import {WorkflowRunArtifactListComponent} from './run/node/artifact/artifact.list.component';
import {WorkflowRunTestsResultComponent} from './run/node/test/tests.component';
import {WorkflowRunTestTableComponent} from './run/node/test/table/test.table.component';
import {WorkflowRunSummaryComponent} from './run/summary/workflow.run.summary.component';
import {WorkflowNodeRunHistoryComponent} from './run/node/history/history.component';
import {WorkflowNodeRunSummaryComponent} from './run/node/summary/run.summary.component';
import {WorkflowAdminComponent} from './show/admin/workflow.admin.component';
import {WorkflowNotificationComponent} from './show/notification/workflow.notification.component';

@NgModule({
    declarations: [
        WorkflowAdminComponent,
        WorkflowComponent,
        WorkflowAddComponent,
        WorkflowBreadCrumbComponent,
        WorkflowGraphComponent,
        WorkflowRunComponent,
        WorkflowNotificationComponent,
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
        WorkflowNodeRunHistoryComponent,
        WorkflowSidebarRunListComponent,
        WorkflowSidebarRunNodeComponent,
        WorkflowShowComponent,
        WorkflowStepLogComponent,
        WorkflowNodeRunSummaryComponent
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
