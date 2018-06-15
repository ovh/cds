import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthAdminRoute} from '../../service/auth/authenAdminRouteActivate';
import {AdminComponent} from './admin.component';
import {BroadcastAddComponent} from './broadcast/add/broadcast.add.component';
import {BroadcastEditComponent} from './broadcast/edit/broadcast.edit.component';
import {BroadcastListComponent} from './broadcast/list/broadcast.list.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {MigrationProjectComponent} from './migration/project/migration.project.component';
import {WorkerModelPatternAddComponent} from './worker-model-pattern/add/worker-model-pattern.add.component';
import {WorkerModelPatternEditComponent} from './worker-model-pattern/edit/worker-model-pattern.edit.component';
import {WorkerModelPatternComponent} from './worker-model-pattern/worker-model-pattern.component';

const routes: Routes = [
    {
        path: '',
        component: AdminComponent,
        canActivateChild: [CanActivateAuthAdminRoute],
        canActivate: [CanActivateAuthAdminRoute],
        children: [
            { path: 'migration', component: MigrationOverviewComponent, data: { title: 'Admin - Migration' } },
            { path: 'migration/:key', component: MigrationProjectComponent },
            {
                path: 'worker-model-pattern',
                component: WorkerModelPatternComponent,
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
            }
        ]
    }
];

export const AdminRouting: ModuleWithProviders = RouterModule.forChild(routes);
