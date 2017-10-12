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
import {WorkflowSidebarComponent} from './sidebar/workflow.sidebar.component';
import {WorkflowRunArtifactListComponent} from './run/node/artifact/artifact.list.component';
import {WorkflowRunTestsResultComponent} from './run/node/test/tests.component';
import {WorkflowRunTestTableComponent} from './run/node/test/table/test.table.component';
import {WorkflowRunSummaryComponent} from './run/summary/workflow.run.summary.component';
import {WorkflowNodeRunHistoryComponent} from './run/node/history/history.component';
import {WorkflowNodeRunSummaryComponent} from './run/node/summary/run.summary.component';
import {WorkflowAdminComponent} from './show/admin/workflow.admin.component';
import {SidebarService} from 'ng2-semantic-ui/dist';

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
        WorkflowNodeRunHistoryComponent,
        WorkflowSidebarComponent,
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
