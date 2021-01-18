import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { MaintainerGuard } from 'app/guard/admin.guard';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { AdminModule } from 'app/views/admin/admin.module';
import { AdminComponent } from './admin.component';
import { BroadcastAddComponent } from './broadcast/add/broadcast.add.component';
import { BroadcastEditComponent } from './broadcast/edit/broadcast.edit.component';
import { BroadcastListComponent } from './broadcast/list/broadcast.list.component';
import { HookTaskListComponent } from './hook-task/list/hook-task.list.component';
import { HookTaskShowComponent } from './hook-task/show/hook-task.show.component';
import { ServiceListComponent } from './service/list/service.list.component';
import { ServiceShowComponent } from './service/show/service.show.component';
import { WorkerModelPatternAddComponent } from './worker-model-pattern/add/worker-model-pattern.add.component';
import { WorkerModelPatternEditComponent } from './worker-model-pattern/edit/worker-model-pattern.edit.component';
import { WorkerModelPatternListComponent } from './worker-model-pattern/list/worker-model-pattern.list.component';

const routes: Routes = [
    {
        path: '',
        component: AdminComponent,
        canActivateChild: [AuthenticationGuard, MaintainerGuard],
        canActivate: [AuthenticationGuard, MaintainerGuard],
        children: [
            {
                path: 'worker-model-pattern',
                component: WorkerModelPatternListComponent,
                data: { title: 'List • Worker Model Pattern' }
            },
            {
                path: 'worker-model-pattern/add',
                component: WorkerModelPatternAddComponent,
                data: { title: 'Add • Worker Model Pattern' }
            },
            {
                path: 'worker-model-pattern/:type/:name',
                component: WorkerModelPatternEditComponent,
                data: { title: '{name} • Edit • Worker Model Pattern' }
            },
            {
                path: 'broadcast',
                component: BroadcastListComponent,
                data: { title: 'List • Broadcast' }
            },
            {
                path: 'broadcast/add',
                component: BroadcastAddComponent,
                data: { title: 'Add • Broadcast' }
            },
            {
                path: 'broadcast/:id',
                component: BroadcastEditComponent,
                data: { title: 'Edit {id} • Broadcast' }
            },
            {
                path: 'hooks-tasks',
                component: HookTaskListComponent,
                data: { title: 'List • Hook task' }
            },
            {
                path: 'hooks-tasks/:id',
                component: HookTaskShowComponent,
                data: { title: 'Show • Hook task' }
            },
            {
                path: 'services',
                component: ServiceListComponent,
                data: { title: 'Services' }
            },
            {
                path: 'services/:name',
                component: ServiceShowComponent,
                data: { title: 'Service' }
            }
        ]
    }
];

export const AdminRouting: ModuleWithProviders<AdminModule> = RouterModule.forChild(routes);
