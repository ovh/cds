import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {AdminComponent} from './admin.component';
import {MigrationOverviewComponent} from './migration/migration.overview.component';
import {StatusComponent} from './status/status.component';
import {CanActivateAuthAdminRoute} from '../../service/auth/authenAdminRouteActivate';
import {MigrationProjectComponent} from './migration/project/migration.project.component';
import {InfoAddComponent} from './info/add/info.add.component';
import {InfoEditComponent} from './info/edit/info.edit.component';
import {InfoListComponent} from './info/list/info.list.component';

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
            { path: 'info', component: InfoListComponent },
            { path: 'info/add', component: InfoAddComponent },
            { path: 'info/:id', component: InfoEditComponent }
        ]
    }
];

export const AdminRouting: ModuleWithProviders = RouterModule.forChild(routes);
