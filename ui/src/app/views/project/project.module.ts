import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {ProjectShowComponent} from './show/project.component';
import {ProjectAddComponent} from './add/project.add.component';
import {projectRouting} from './project.routing';
import {SharedModule} from '../../shared/shared.module';
import {ProjectAdminComponent} from './show/admin/project.admin.component';
import {ProjectRepoManagerComponent} from './show/admin/repomanager/list/project.repomanager.list.component';
import {ProjectRepoManagerFormComponent} from './show/admin/repomanager/form/project.repomanager.form.component';
import {RouterModule} from '@angular/router';
import {ProjectEnvironmentListComponent} from './show/environment/list/environment.list.component';
import {ProjectEnvironmentComponent} from './show/environment/list/item/environment.component';
import {ProjectEnvironmentFormComponent} from './show/environment/form/environment.form.component';

@NgModule({
    declarations: [
        ProjectShowComponent,
        ProjectAddComponent,
        ProjectAdminComponent,
        ProjectRepoManagerComponent,
        ProjectRepoManagerFormComponent,
        ProjectEnvironmentFormComponent,
        ProjectEnvironmentListComponent,
        ProjectEnvironmentComponent
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
