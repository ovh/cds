import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {AdminComponent} from './admin.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {WorkerModelPatternComponent} from './worker-model-pattern/worker-model-pattern.component';
import {WorkerModelPatternAddComponent} from './worker-model-pattern/add/worker-model-pattern.add.component';
import {WorkerModelPatternEditComponent} from './worker-model-pattern/edit/worker-model-pattern.edit.component';
import {CanActivateAuthAdminRoute} from '../../service/auth/authenAdminRouteActivate';
import {MigrationProjectComponent} from './migration/project/migration.project.component';
import {BroadcastAddComponent} from './broadcast/add/broadcast.add.component';
import {BroadcastEditComponent} from './broadcast/edit/broadcast.edit.component';
import {BroadcastListComponent} from './broadcast/list/broadcast.list.component';

const routes: Routes = [
    {
        path: '',
        component: AdminComponent,
        canActivateChild: [CanActivateAuthAdminRoute],
        canActivate: [CanActivateAuthAdminRoute],
        children: [
            { path: 'migration', component: MigrationOverviewComponent, data: { title: 'CDS - Admin - Migration' } },
            { path: 'migration/:key', component: MigrationProjectComponent },
            {
                path: 'worker-model-pattern',
                component: WorkerModelPatternComponent,
                data: { title: 'CDS - Admin - Worker Model Pattern - List' }
            },
            {
                path: 'worker-model-pattern/add',
                component: WorkerModelPatternAddComponent,
                data: { title: 'CDS - Admin - Worker Model Pattern - Add' }
            },
            {
                path: 'worker-model-pattern/:type/:name',
                component: WorkerModelPatternEditComponent,
                data: { title: 'CDS - Admin - Worker Model Pattern - Edit {name}' }
            },
            {
                path: 'broadcast',
                component: BroadcastListComponent,
                data: { title: 'CDS - Admin - Broadcast - List' }
            },
            {
                path: 'broadcast/add',
                component: BroadcastAddComponent,
                data: { title: 'CDS - Admin - Broadcast - Add' }
            },
            {
                path: 'broadcast/:id',
                component: BroadcastEditComponent,
                data: { title: 'CDS - Admin - Broadcast - Edit {id}' }
            }
        ]
    }
];

export const AdminRouting: ModuleWithProviders = RouterModule.forChild(routes);
