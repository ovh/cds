import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { MaintainerGuard } from 'app/guard/admin.guard';
import { AdminModule } from 'app/views/admin/admin.module';
import { AdminComponent } from './admin.component';
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
        canActivateChild: [MaintainerGuard],
        canActivate: [MaintainerGuard],
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
