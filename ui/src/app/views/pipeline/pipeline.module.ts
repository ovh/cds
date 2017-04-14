import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {pipelineRouting} from './pipeline.routing';
import {SharedModule} from '../../shared/shared.module';
import {PipelineShowComponent} from './show/pipeline.show.component';
import {PipelineWorkflowComponent} from './show/workflow/pipeline.workflow.component';
import {PipelineStageComponent} from './show/workflow/stage/pipeline.stage.component';
import {PipelineAdminComponent} from './show/admin/pipeline.admin.component';
import {PipelineAddComponent} from './add/pipeline.add.component';
import {PipelineStageFormComponent} from './show/workflow/stage/form/pipeline.stage.form.component';
import {PipelineApplicationComponent} from './show/application/pipeline.application.component';

@NgModule({
    declarations: [
        PipelineApplicationComponent,
        PipelineShowComponent,
        PipelineAddComponent,
        PipelineWorkflowComponent,
        PipelineStageComponent,
        PipelineStageFormComponent,
        PipelineAdminComponent
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
