import {CUSTOM_ELEMENTS_SCHEMA, NgModule} from '@angular/core';
import {RouterModule} from '@angular/router';
import {SharedModule} from '../../shared/shared.module';
import {ProjectAddComponent} from './add/project.add.component';
import {ProjectListComponent} from './list/project.list.component';
import {projectRouting} from './project.routing';
import {ProjectAdminComponent} from './show/admin/project.admin.component';
import {ProjectRepoManagerComponent} from './show/admin/repomanager/list/project.repomanager.list.component';
import {ProjectApplicationListComponent} from './show/application/application.list.component';
import {ProjectEnvironmentFormComponent} from './show/environment/form/environment.form.component';
import {ProjectEnvironmentListComponent} from './show/environment/list/environment.list.component';
import {ProjectEnvironmentComponent} from './show/environment/list/item/environment.component';
import {ProjectKeysComponent} from './show/keys/project.keys.component';
import {ProjectPermissionsComponent} from './show/permission/permission.component';
import {ProjectPipelinesComponent} from './show/pipeline/pipeline.list.component';
import {ProjectPlatformFormComponent} from './show/platforms/form/platform.form.component';
import {ProjectPlatformListComponent} from './show/platforms/list/platform.list.component';
import {ProjectPlatformsComponent} from './show/platforms/project.platforms.component';
import {ProjectShowComponent} from './show/project.component';
import {ProjectVariablesComponent} from './show/variable/variable.list.component';
import {ProjectWorkflowListComponent} from './show/workflow/workflow.list.component';

@NgModule({
    declarations: [
        ProjectAddComponent,
        ProjectListComponent,
        ProjectAdminComponent,
        ProjectApplicationListComponent,
        ProjectEnvironmentFormComponent,
        ProjectEnvironmentListComponent,
        ProjectEnvironmentComponent,
        ProjectKeysComponent,
        ProjectPipelinesComponent,
        ProjectVariablesComponent,
        ProjectPermissionsComponent,
        ProjectRepoManagerComponent,
        ProjectShowComponent,
        ProjectWorkflowListComponent,
        ProjectPlatformsComponent,
        ProjectPlatformFormComponent,
        ProjectPlatformListComponent
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
