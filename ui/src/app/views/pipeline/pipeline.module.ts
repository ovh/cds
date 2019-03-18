import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { SharedModule } from '../../shared/shared.module';
import { PipelineAddComponent } from './add/pipeline.add.component';
import { pipelineRouting } from './pipeline.routing';
import { PipelineAdminComponent } from './show/admin/pipeline.admin.component';
import { PipelineAsCodeEditorComponent } from './show/ascode-editor/pipeline.ascode.editor.component';
import { PipelineAuditComponent } from './show/audit/pipeline.audit.component';
import { PipelineShowComponent } from './show/pipeline.show.component';
import { PipelineWorkflowComponent } from './show/workflow/pipeline.workflow.component';
import { PipelineStageFormComponent } from './show/workflow/stage/form/pipeline.stage.form.component';
@NgModule({
    declarations: [
        PipelineShowComponent,
        PipelineAddComponent,
        PipelineWorkflowComponent,
        PipelineStageFormComponent,
        PipelineAuditComponent,
        PipelineAdminComponent,
        PipelineAsCodeEditorComponent,
    ],
    imports: [
        SharedModule,
        pipelineRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class PipelineModule {
}
