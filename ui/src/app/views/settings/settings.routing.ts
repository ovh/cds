import {Routes, RouterModule} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ModuleWithProviders} from '@angular/core';
import {SettingsComponent} from './settings.component';
import {ActionAddComponent} from './action/add/action.add.component';
import {ActionEditComponent} from './action/edit/action.edit.component';
import {ActionListComponent} from './action/list/action.list.component';
import {DownloadComponent} from './download/download.component';
import {GroupEditComponent} from './group/edit/group.edit.component';
import {GroupListComponent} from './group/list/group.list.component';
import {UserEditComponent} from './user/edit/user.edit.component';
import {UserListComponent} from './user/list/user.list.component';
import {QueueComponent} from './queue/queue.component';
import {StatusComponent} from './status/status.component';
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
            { path: 'profile/:username', component: UserEditComponent, data: { title: 'CDS - Profile' }},
            { path: 'worker-model', component: WorkerModelListComponent, data: { title: 'CDS - Worker Model' } },
            { path: 'worker-model/add', component: WorkerModelAddComponent, data: { title: 'CDS - Worker Model - Add' } },
            {
                path: 'worker-model/:workerModelName',
                component: WorkerModelEditComponent,
                data: { title: 'CDS - Worker Model {workerModelName}' }
            },
            { path: 'group', component: GroupListComponent, data: { title: 'CDS - Groups' } },
            { path: 'group/:groupname', component: GroupEditComponent, data: { title: 'CDS - Group {groupname}' } },
            { path: 'user', component: UserListComponent, data: { title: 'CDS - User list' } },
            { path: 'user/:username', component: UserEditComponent, data: { title: 'CDS - User {username}' } },
            { path: 'action', component: ActionListComponent, data: { title: 'CDS - Actions' } },
            { path: 'action/add', component: ActionAddComponent, data: { title: 'CDS - Action - Add' } },
            { path: 'action/:name', component: ActionEditComponent, data: { title: 'CDS - Action {name}' } },
            { path: 'queue', component: QueueComponent, data: { title: 'CDS - Queue' }},
            { path: 'status', component: StatusComponent, data: { title: 'CDS - Status' } },
            { path: 'downloads', component: DownloadComponent, data: { title: 'CDS - Downloads' } }
        ]
    }
];

export const SettingsRouting: ModuleWithProviders = RouterModule.forChild(routes);
