import { CUSTOM_ELEMENTS_SCHEMA, NgModule, NO_ERRORS_SCHEMA } from '@angular/core';
import { SharedModule } from '../../shared/shared.module';
import { WorkflowAddComponent } from './add/workflow.add.component';
import { WorkflowBreadCrumbComponent } from './breadcrumb/workflow.breadcrumb.component';
import { WorkflowGraphComponent } from './graph/workflow.graph.component';
import { WorkflowRunArtifactListComponent } from './run/node/artifact/artifact.list.component';
import { WorkflowNodeRunHistoryComponent } from './run/node/history/history.component';
import { WorkflowRunNodePipelineComponent } from './run/node/pipeline/node.pipeline.component';
import { WorkflowNodeRunSummaryComponent } from './run/node/summary/run.summary.component';
import { WorkflowRunTestTableComponent } from './run/node/test/table/test.table.component';
import { WorkflowRunTestsResultComponent } from './run/node/test/tests.component';
import { WorkflowNodeRunComponent } from './run/node/workflow.run.node.component';
import { WorkflowRunSummaryComponent } from './run/summary/workflow.run.summary.component';
import { WorkflowRunComponent } from './run/workflow.run.component';
import { WorkflowAdminComponent } from './show/admin/workflow.admin.component';
import { WorkflowNotificationFormComponent } from './show/notification/form/workflow.notification.form.component';
import { WorkflowNotificationListComponent } from './show/notification/list/workflow.notification.list.component';
import { WorkflowShowComponent } from './show/workflow.component';
import { WorkflowSidebarCodeComponent } from './sidebar/code/sidebar.code.component';
import { WorkflowComponent } from './workflow.component';
import { workflowRouting } from './workflow.routing';
import { WorkflowDeleteModalComponent } from './show/admin/delete-modal/delete-modal.component';

@NgModule({
    declarations: [
        WorkflowAddComponent,
        WorkflowAdminComponent,
        WorkflowBreadCrumbComponent,
        WorkflowComponent,
        WorkflowGraphComponent,
        WorkflowNodeRunComponent,
        WorkflowNodeRunHistoryComponent,
        WorkflowNodeRunSummaryComponent,
        WorkflowNotificationFormComponent,
        WorkflowNotificationListComponent,
        WorkflowRunArtifactListComponent,
        WorkflowRunComponent,
        WorkflowRunNodePipelineComponent,
        WorkflowRunSummaryComponent,
        WorkflowRunTestsResultComponent,
        WorkflowRunTestTableComponent,
        WorkflowShowComponent,
        WorkflowSidebarCodeComponent,
        WorkflowDeleteModalComponent
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
