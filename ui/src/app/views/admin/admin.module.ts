import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {RouterModule} from '@angular/router';
import {AdminRouting} from './admin.routing';
import {AdminComponent} from './admin.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {MigrationProjectComponent} from './migration/project/migration.project.component';
import {WorkerModelPatternComponent} from './worker-model-pattern/worker-model-pattern.component';
import {WorkerModelPatternAddComponent} from './worker-model-pattern/add/worker-model-pattern.add.component';
import {WorkerModelPatternEditComponent} from './worker-model-pattern/edit/worker-model-pattern.edit.component';
import {StatusComponent} from './status/status.component';
import {AdminSidebarComponent} from './sidebar/admin.sidebar.component';

@NgModule({
    declarations: [
        AdminComponent,
        AdminSidebarComponent,
        MigrationOverviewComponent,
        MigrationProjectComponent,
        StatusComponent,
        WorkerModelPatternComponent,
        WorkerModelPatternAddComponent,
        WorkerModelPatternEditComponent,
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
