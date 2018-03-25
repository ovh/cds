import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {RouterModule} from '@angular/router';
import {AdminRouting} from './admin.routing';
import {AdminComponent} from './admin.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {MigrationProjectComponent} from './migration/project/migration.project.component';
import {StatusComponent} from './status/status.component';
import {InfoAddComponent} from './info/add/info.add.component';
import {InfoEditComponent} from './info/edit/info.edit.component';
import {InfoListComponent} from './info/list/info.list.component';

@NgModule({
    declarations: [
        AdminComponent,
        MigrationOverviewComponent,
        MigrationProjectComponent,
        StatusComponent,
        InfoAddComponent,
        InfoEditComponent,
        InfoListComponent
    ],
    imports: [
      SharedModule,
      RouterModule,
      AdminRouting
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class AdminModule {
}
