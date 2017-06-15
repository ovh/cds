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

@NgModule({
    declarations: [
        WorkflowAddComponent,
        WorkflowBreadCrumbComponent,
        WorkflowGraphComponent,
        WorkflowRunComponent,
        WorkflowNodeRunComponent,
        WorkflowRunNodePipelineComponent,
        WorkflowShowComponent
    ],
    imports: [
        SharedModule,
        workflowRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA,
        NO_ERRORS_SCHEMA
    ]
})
export class WorkflowModule {
}
