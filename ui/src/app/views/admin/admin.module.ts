import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { MaintainerGuard } from 'app/guard/admin.guard';
import { SharedModule } from '../../shared/shared.module';
import { AdminComponent } from './admin.component';
import { AdminRouting } from './admin.routing';
import { BroadcastAddComponent } from './broadcast/add/broadcast.add.component';
import { BroadcastEditComponent } from './broadcast/edit/broadcast.edit.component';
import { BroadcastListComponent } from './broadcast/list/broadcast.list.component';
import { HookTaskListComponent } from './hook-task/list/hook-task.list.component';
import { HookTaskShowComponent } from './hook-task/show/hook-task.show.component';
import { ServiceListComponent } from './service/list/service.list.component';
import { ServiceShowComponent } from './service/show/service.show.component';
import { WorkerModelPatternAddComponent } from './worker-model-pattern/add/worker-model-pattern.add.component';
import { WorkerModelPatternEditComponent } from './worker-model-pattern/edit/worker-model-pattern.edit.component';
import { WorkerModelPatternFormComponent } from './worker-model-pattern/form/worker-model.pattern.form.component';
import { WorkerModelPatternListComponent } from './worker-model-pattern/list/worker-model-pattern.list.component';

@NgModule({
    declarations: [
        AdminComponent,
        WorkerModelPatternListComponent,
        WorkerModelPatternAddComponent,
        WorkerModelPatternEditComponent,
        WorkerModelPatternFormComponent,
        BroadcastAddComponent,
        BroadcastEditComponent,
        BroadcastListComponent,
        HookTaskListComponent,
        HookTaskShowComponent,
        ServiceListComponent,
        ServiceShowComponent
    ],
    imports: [
        SharedModule,
        RouterModule,
        AdminRouting
    ],
    providers: [
        MaintainerGuard,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class AdminModule {
}
