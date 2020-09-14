import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { SettingsModule } from 'app/views/settings/settings.module';
import { ActionAddComponent } from './action/add/action.add.component';
import { ActionEditComponent } from './action/edit/action.edit.component';
import { ActionListComponent } from './action/list/action.list.component';
import { ActionShowComponent } from './action/show/action.show.component';
import { CdsctlComponent } from './cdsctl/cdsctl.component';
import { DownloadComponent } from './download/download.component';
import { GroupEditComponent } from './group/edit/group.edit.component';
import { GroupListComponent } from './group/list/group.list.component';
import { QueueComponent } from './queue/queue.component';
import { SettingsComponent } from './settings.component';
import { UserEditComponent } from './user/edit/user.edit.component';
import { UserListComponent } from './user/list/user.list.component';
import { WorkerModelAddComponent } from './worker-model/add/worker-model.add.component';
import { WorkerModelEditComponent } from './worker-model/edit/worker-model.edit.component';
import { WorkerModelListComponent } from './worker-model/list/worker-model.list.component';
import { WorkflowTemplateAddComponent } from './workflow-template/add/workflow-template.add.component';
import { WorkflowTemplateEditComponent } from './workflow-template/edit/workflow-template.edit.component';
import { WorkflowTemplateListComponent } from './workflow-template/list/workflow-template.list.component';

const routes: Routes = [
    {
        path: '',
        component: SettingsComponent,
        canActivateChild: [AuthenticationGuard],
        canActivate: [AuthenticationGuard],
        children: [
            { path: 'profile/:username', component: UserEditComponent, data: { title: 'Profile' } },
            { path: 'cdsctl', component: CdsctlComponent, data: { title: 'Cdsctl' } },
            { path: 'worker-model', component: WorkerModelListComponent, data: { title: 'Worker model' } },
            { path: 'worker-model/add', component: WorkerModelAddComponent, data: { title: 'Add • Worker model' } },
            {
                path: 'worker-model/:groupName/:workerModelName',
                component: WorkerModelEditComponent,
                data: { title: '{workerModelName} • Worker model' }
            },
            { path: 'group', component: GroupListComponent, data: { title: 'Groups' } },
            { path: 'group/:groupname', component: GroupEditComponent, data: { title: '{groupname} • Group' } },
            { path: 'user', component: UserListComponent, data: { title: 'List • User' } },
            { path: 'user/:username', component: UserEditComponent, data: { title: '{username} • User' } },
            { path: 'action', component: ActionListComponent, data: { title: 'Actions' } },
            { path: 'action/add', component: ActionAddComponent, data: { title: 'Add • Action' } },
            {
                path: 'action/:groupName/:actionName',
                component: ActionEditComponent,
                data: { title: '{actionName} • Action' }
            },
            {
                path: 'action-builtin/:actionName',
                component: ActionShowComponent,
                data: { title: '{actionName} • Action' }
            },
            { path: 'queue', component: QueueComponent, data: { title: 'Queue' } },
            { path: 'downloads', component: DownloadComponent, data: { title: 'Downloads' } },
            { path: 'workflow-template', component: WorkflowTemplateListComponent, data: { title: 'Workflow template' } },
            { path: 'workflow-template/add', component: WorkflowTemplateAddComponent, data: { title: 'Add • Workflow template' } },
            {
                path: 'workflow-template/:groupName/:templateSlug',
                component: WorkflowTemplateEditComponent,
                data: { title: '{templateSlug} • Workflow template' }
            },
        ]
    }
];

export const SettingsRouting: ModuleWithProviders<SettingsModule> = RouterModule.forChild(routes);
