import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from 'app/shared/shared.module';
import { ProjectAddComponent } from './add/project.add.component';
import { ProjectListComponent } from './list/project.list.component';
import { projectRouting } from './project.routing';
import { ProjectAdminComponent } from './show/admin/project.admin.component';
import { ProjectRepoManagerComponent } from './show/admin/repomanager/list/project.repomanager.list.component';
import { ProjectApplicationListComponent } from './show/application/application.list.component';
import { ProjectEnvironmentListComponent } from './show/environment/environment.list.component';
import { ProjectIntegrationFormComponent } from './show/integrations/form/integration.form.component';
import { ProjectIntegrationListComponent } from './show/integrations/list/integration.list.component';
import { ProjectIntegrationsComponent } from './show/integrations/project.integrations.component';
import { ProjectKeysComponent } from './show/keys/project.keys.component';
import { ProjectPermissionsComponent } from './show/permission/permission.component';
import { ProjectPipelinesComponent } from './show/pipeline/pipeline.list.component';
import { ProjectShowComponent } from './show/project.component';
import { ProjectVariablesComponent } from './show/variable/variable.list.component';
import { ProjectWarningsComponent } from './show/warnings/warnings.component';
import { ProjectWorkflowListBlocsComponent } from './show/workflow/blocs/workflow.list.blocs.component';
import { ProjectWorkflowListLabelsComponent } from './show/workflow/labels/workflow.list.labels.component';
import { ProjectWorkflowListLinesComponent } from './show/workflow/lines/workflow.list.lines.component';
import { ProjectWorkflowListComponent } from './show/workflow/workflow.list.component';

@NgModule({
    declarations: [
        ProjectAddComponent,
        ProjectListComponent,
        ProjectAdminComponent,
        ProjectApplicationListComponent,
        ProjectEnvironmentListComponent,
        ProjectKeysComponent,
        ProjectPipelinesComponent,
        ProjectVariablesComponent,
        ProjectPermissionsComponent,
        ProjectRepoManagerComponent,
        ProjectShowComponent,
        ProjectWorkflowListBlocsComponent,
        ProjectWorkflowListLabelsComponent,
        ProjectWorkflowListLinesComponent,
        ProjectWorkflowListComponent,
        ProjectEnvironmentListComponent,
        ProjectIntegrationsComponent,
        ProjectIntegrationFormComponent,
        ProjectIntegrationListComponent,
        ProjectWarningsComponent
    ],
    imports: [
        SharedModule,
        RouterModule,
        projectRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectModule {
}
