import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {AdminComponent} from './admin.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {StatusComponent} from './status/status.component';
import {QueueComponent} from './queue/queue.component';
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
            { path: 'migration', component: MigrationOverviewComponent },
            { path: 'migration/:key', component: MigrationProjectComponent },
            { path: 'status', component: StatusComponent },
            { path: 'queue', component: QueueComponent },
            { path: 'worker-model-pattern', component: WorkerModelPatternComponent },
            { path: 'worker-model-pattern/add', component: WorkerModelPatternAddComponent },
            { path: 'worker-model-pattern/:type/:name', component: WorkerModelPatternEditComponent },
            { path: 'broadcast', component: BroadcastListComponent },
            { path: 'broadcast/add', component: BroadcastAddComponent },
            { path: 'broadcast/:id', component: BroadcastEditComponent }
        ]
    }
];

export const AdminRouting: ModuleWithProviders = RouterModule.forChild(routes);
