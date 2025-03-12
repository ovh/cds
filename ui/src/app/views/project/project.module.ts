import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from 'app/shared/shared.module';
import { ProjectAddComponent } from './add/project.add.component';
import { projectRouting } from './project.routing';
import { ProjectRepoManagerListComponent } from './settings/advanced/repomanager/list/project.repomanager.list.component';
import { ProjectApplicationListComponent } from './show/application/application.list.component';
import { ProjectEnvironmentListComponent } from './show/environment/environment.list.component';
import { ProjectIntegrationsComponent } from './settings/integrations/project.integrations.component';
import { ProjectKeysComponent } from './settings/keys/project.keys.component';
import { ProjectPermissionsComponent } from './show/permission/permission.component';
import { ProjectPipelinesComponent } from './show/pipeline/pipeline.list.component';
import { ProjectShowComponent } from './show/project.component';
import { ProjectVariablesComponent } from './show/variable/variable.list.component';
import { ProjectWorkflowListBlocsComponent } from './show/workflow/blocs/workflow.list.blocs.component';
import { ProjectWorkflowListLabelsComponent } from './show/workflow/labels/workflow.list.labels.component';
import { ProjectWorkflowListLinesComponent } from './show/workflow/lines/workflow.list.lines.component';
import { ProjectWorkflowListComponent } from './show/workflow/workflow.list.component';
import { ProjectComponent } from './project.component';
import { ProjectActivityBarComponent } from './activity-bar/activity-bar.component';
import { ProjectSettingsComponent } from './settings/settings.component';
import { ProjectVariableSetsComponent } from './settings/variablesets/variablesets.component';
import { ProjectVariableSetItemsComponent } from './settings/variablesets/items/variableset.item.component';
import { ProjectAdvancedComponent } from './settings/advanced/project.advanced.component';
import { ProjectRepoManagerFormComponent } from './settings/advanced/repomanager/from/project.repomanager.form.component';
import { ProjectExistsGuard, ProjectGuard, ProjectV2Guard } from './project.guard';
import { ProjectConcurrenciesComponent } from './settings/concurrency/concurrencies.components';
import { ProjectConcurrencyFormComponent } from './settings/concurrency/concurrency.form.component';

@NgModule({
    declarations: [
        ProjectActivityBarComponent,
        ProjectAddComponent,
        ProjectAdvancedComponent,
        ProjectApplicationListComponent,
        ProjectComponent,
        ProjectConcurrenciesComponent,
        ProjectConcurrencyFormComponent,
        ProjectEnvironmentListComponent,
        ProjectEnvironmentListComponent,
        ProjectIntegrationsComponent,
        ProjectKeysComponent,
        ProjectPermissionsComponent,
        ProjectPipelinesComponent,
        ProjectRepoManagerFormComponent,
        ProjectRepoManagerListComponent,
        ProjectSettingsComponent,
        ProjectShowComponent,
        ProjectVariablesComponent,
        ProjectVariableSetItemsComponent,
        ProjectVariableSetsComponent,
        ProjectWorkflowListBlocsComponent,
        ProjectWorkflowListComponent,
        ProjectWorkflowListLabelsComponent,
        ProjectWorkflowListLinesComponent
    ],
    imports: [
        SharedModule,
        RouterModule,
        projectRouting
    ],
    providers: [
        ProjectExistsGuard,
        ProjectGuard,
        ProjectV2Guard
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectModule { }
