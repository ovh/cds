import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {AdminComponent} from './admin.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {StatusComponent} from './status/status.component';
import {WorkerModelPatternComponent} from './worker-model-pattern/worker-model-pattern.component';
import {WorkerModelPatternAddComponent} from './worker-model-pattern/add/worker-model-pattern.add.component';
import {WorkerModelPatternEditComponent} from './worker-model-pattern/edit/worker-model-pattern.edit.component';
import {CanActivateAuthAdminRoute} from '../../service/auth/authenAdminRouteActivate';
import {MigrationProjectComponent} from './migration/project/migration.project.component';
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
            { path: 'worker-model-pattern', component: WorkerModelPatternComponent },
            { path: 'worker-model-pattern/add', component: WorkerModelPatternAddComponent },
            { path: 'worker-model-pattern/:type/:name', component: WorkerModelPatternEditComponent },
        ]
    }
];

export const AdminRouting: ModuleWithProviders = RouterModule.forChild(routes);
