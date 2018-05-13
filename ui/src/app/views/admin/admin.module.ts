import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {RouterModule} from '@angular/router';
import {AdminRouting} from './admin.routing';
import {AdminComponent} from './admin.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {MigrationProjectComponent} from './migration/project/migration.project.component';
import {StatusComponent} from './status/status.component';
import {BroadcastAddComponent} from './broadcast/add/broadcast.add.component';
import {BroadcastEditComponent} from './broadcast/edit/broadcast.edit.component';
import {BroadcastListComponent} from './broadcast/list/broadcast.list.component';

@NgModule({
    declarations: [
        AdminComponent,
        MigrationOverviewComponent,
        MigrationProjectComponent,
        StatusComponent,
        BroadcastAddComponent,
        BroadcastEditComponent,
        BroadcastListComponent
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
