import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {PipelineRunWorkflowComponent} from './workflow/workflow.component';
import {ArtifactListComponent} from './artifact/artifact.list.component';
import {applicationRunRouting} from './application.run.routing';
import {ApplicationPipelineBuildComponent} from './pipeline.build.component';
import {StepLogComponent} from './workflow/step/step.log.component';
import {TestsResultComponent} from './test/tests.component';
import {TestTableComponent} from './test/table/test.table.component';
import {SharedModule} from '../../shared/shared.module';
import {RunSummaryComponent} from './summary/run.summary.component';
import {SpawnInfoComponent} from './workflow/spwaninfo/spawninfo.component';
import {JobVariableComponent} from './workflow/variables/job.variables.component';

@NgModule({
    declarations: [
        ApplicationPipelineBuildComponent,
        PipelineRunWorkflowComponent,
        ArtifactListComponent,
        JobVariableComponent,
        RunSummaryComponent,
        SpawnInfoComponent,
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
