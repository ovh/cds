import {CUSTOM_ELEMENTS_SCHEMA, NgModule} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {applicationRunRouting} from './application.run.routing';
import {ArtifactListComponent} from './artifact/artifact.list.component';
import {ApplicationPipelineBuildComponent} from './pipeline.build.component';
import {RunSummaryComponent} from './summary/run.summary.component';
import {TestTableComponent} from './test/table/test.table.component';
import {TestsResultComponent} from './test/tests.component';
import {SpawnInfoComponent} from './workflow/spwaninfo/spawninfo.component';
import {StepLogComponent} from './workflow/step/step.log.component';
import {JobVariableComponent} from './workflow/variables/job.variables.component';
import {PipelineRunWorkflowComponent} from './workflow/workflow.component';

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
