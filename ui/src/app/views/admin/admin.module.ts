import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from '../../shared/shared.module';
import { AdminComponent } from './admin.component';
import { AdminRouting } from './admin.routing';
import { BroadcastAddComponent } from './broadcast/add/broadcast.add.component';
import { BroadcastEditComponent } from './broadcast/edit/broadcast.edit.component';
import { BroadcastListComponent } from './broadcast/list/broadcast.list.component';
import { HooksTaskComponent } from './hooks-tasks/details/hooks-task.component';
import { HooksTasksComponent } from './hooks-tasks/hooks-tasks.component';
import { ServiceComponent } from './services/service/service.component';
import { ServicesComponent } from './services/services.component';
import { WorkerModelPatternAddComponent } from './worker-model-pattern/add/worker-model-pattern.add.component';
import { WorkerModelPatternEditComponent } from './worker-model-pattern/edit/worker-model-pattern.edit.component';
import { WorkerModelPatternComponent } from './worker-model-pattern/worker-model-pattern.component';

@NgModule({
    declarations: [
        AdminComponent,
        WorkerModelPatternComponent,
        WorkerModelPatternAddComponent,
        WorkerModelPatternEditComponent,
        BroadcastAddComponent,
        BroadcastEditComponent,
        BroadcastListComponent,
        HooksTasksComponent,
        HooksTaskComponent,
        ServicesComponent,
        ServiceComponent
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
