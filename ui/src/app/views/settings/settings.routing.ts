import {Routes, RouterModule} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ModuleWithProviders} from '@angular/core';
import {SettingsComponent} from './settings.component';
import {ActionEditComponent} from './action/edit/action.edit.component';
import {ActionListComponent} from './action/list/action.list.component';
import {GroupEditComponent} from './group/edit/group.edit.component';
import {GroupListComponent} from './group/list/group.list.component';
import {UserEditComponent} from './user/edit/user.edit.component';
import {UserListComponent} from './user/list/user.list.component';
import {WorkerModelEditComponent} from './worker-model/edit/worker-model.edit.component';
import {WorkerModelListComponent} from './worker-model/list/worker-model.list.component';

const routes: Routes = [
    {
        path: '',
        component: SettingsComponent,
        canActivateChild: [CanActivateAuthRoute],
        canActivate: [CanActivateAuthRoute],
        children: [
            { path: 'profile/:username', component: UserEditComponent },
            { path: 'worker-model', component: WorkerModelListComponent },
            { path: 'worker-model/:workerModelName', component: WorkerModelEditComponent },
            { path: 'group', component: GroupListComponent },
            { path: 'group/:groupname', component: GroupEditComponent },
            { path: 'user', component: UserListComponent },
            { path: 'user/:username', component: UserEditComponent },
            { path: 'action', component: ActionListComponent },
            { path: 'action/:name', component: ActionEditComponent }
        ]
    }
];

export const SettingsRouting: ModuleWithProviders = RouterModule.forChild(routes);
