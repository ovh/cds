import {CUSTOM_ELEMENTS_SCHEMA, NgModule, NO_ERRORS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {workflowRouting} from './workflow.routing';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowNodeItemFormComponent} from './add/node-form/workflow.node.form.component';
import {ApplicationWorkflowItemComponent} from '../application/show/workflow/tree/item/application.workflow.item.component';

@NgModule({
    declarations: [
        WorkflowAddComponent,
        WorkflowNodeItemFormComponent,
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
