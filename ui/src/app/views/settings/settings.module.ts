import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {RouterModule} from '@angular/router';
import {SettingsRouting} from './settings.routing';
import {SettingsComponent} from './settings.component';
import {SettingsSidebarComponent} from './sidebar/settings-sidebar.component';
import {ActionEditComponent} from './action/edit/action.edit.component';
import {ActionListComponent} from './action/list/action.list.component';
import {GroupEditComponent} from './group/edit/group.edit.component';
import {GroupListComponent} from './group/list/group.list.component';
import {UserEditComponent} from './user/edit/user.edit.component';
import {UserListComponent} from './user/list/user.list.component';
import {WorkerModelEditComponent} from './worker-model/edit/worker-model.edit.component';
import {WorkerModelListComponent} from './worker-model/list/worker-model.list.component';

@NgModule({
    declarations: [
        SettingsComponent,
        SettingsSidebarComponent,
        ActionEditComponent,
        ActionListComponent,
        GroupEditComponent,
        GroupListComponent,
        UserEditComponent,
        UserListComponent,
        WorkerModelEditComponent,
        WorkerModelListComponent
    ],
    imports: [
      SharedModule,
      RouterModule,
      SettingsRouting
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class SettingsModule {
}
