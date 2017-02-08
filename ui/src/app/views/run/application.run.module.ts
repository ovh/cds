import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {PipelineRunWorkflowComponent} from './workflow/workflow.component';
import {ArtifactListComponent} from './artifact/artifact.list.component';
import {CommitListComponent} from './commit/commit.list.component';
import {applicationRunRouting} from './application.run.routing';
import {ApplicationPipelineBuildComponent} from './pipeline.build.component';
import {StepLogComponent} from './workflow/step/step.log.component';
import {TestsResultComponent} from './test/tests.component';
import {TestTableComponent} from './test/table/test.table.component';
import {SharedModule} from '../../shared/shared.module';

@NgModule({
    declarations: [
        ApplicationPipelineBuildComponent,
        PipelineRunWorkflowComponent,
        ArtifactListComponent,
        CommitListComponent,
        StepLogComponent,
        TestsResultComponent,
        TestTableComponent
    ],
    imports: [
        SharedModule,
        applicationRunRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ApplicationRunModule {
}
