import {CUSTOM_ELEMENTS_SCHEMA, NgModule} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {workflowRouting} from './workflow.routing';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowNodeItemFormComponent} from './add/node-form/workflow.node.form.component';

@NgModule({
    declarations: [
        WorkflowAddComponent,
        WorkflowNodeItemFormComponent,
        WorkflowShowComponent,
    ],
    imports: [
        SharedModule,
        workflowRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class WorkflowModule {
}
