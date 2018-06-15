import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ActionAddComponent} from './action/add/action.add.component';
import {ActionEditComponent} from './action/edit/action.edit.component';
import {ActionListComponent} from './action/list/action.list.component';
import {DownloadComponent} from './download/download.component';
import {GroupEditComponent} from './group/edit/group.edit.component';
import {GroupListComponent} from './group/list/group.list.component';
import {QueueComponent} from './queue/queue.component';
import {SettingsComponent} from './settings.component';
import {StatusComponent} from './status/status.component';
import {UserEditComponent} from './user/edit/user.edit.component';
import {UserListComponent} from './user/list/user.list.component';
import {WorkerModelAddComponent} from './worker-model/add/worker-model.add.component';
import {WorkerModelEditComponent} from './worker-model/edit/worker-model.edit.component';
import {WorkerModelListComponent} from './worker-model/list/worker-model.list.component';

const routes: Routes = [
    {
        path: '',
        component: SettingsComponent,
        canActivateChild: [CanActivateAuthRoute],
        canActivate: [CanActivateAuthRoute],
        children: [
            { path: 'profile/:username', component: UserEditComponent, data: { title: 'Profile' }},
            { path: 'worker-model', component: WorkerModelListComponent, data: { title: 'Worker Model' } },
            { path: 'worker-model/add', component: WorkerModelAddComponent, data: { title: 'Add • Worker Model' } },
            {
                path: 'worker-model/:workerModelName',
                component: WorkerModelEditComponent,
                data: { title: 'Worker Model {workerModelName}' }
            },
            { path: 'group', component: GroupListComponent, data: { title: 'Groups' } },
            { path: 'group/:groupname', component: GroupEditComponent, data: { title: '{groupname} • Group' } },
            { path: 'user', component: UserListComponent, data: { title: 'List • User' } },
            { path: 'user/:username', component: UserEditComponent, data: { title: '{username} • User' } },
            { path: 'action', component: ActionListComponent, data: { title: 'Actions' } },
            { path: 'action/add', component: ActionAddComponent, data: { title: 'Add • Action' } },
            { path: 'action/:name', component: ActionEditComponent, data: { title: '{name} • Action' } },
            { path: 'queue', component: QueueComponent, data: { title: 'Queue' }},
            { path: 'status', component: StatusComponent, data: { title: 'Engine Status' } },
            { path: 'downloads', component: DownloadComponent, data: { title: 'Downloads' } }
        ]
    }
];

export const SettingsRouting: ModuleWithProviders = RouterModule.forChild(routes);
