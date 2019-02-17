import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from '../../shared/shared.module';
import { ActionAddComponent } from './action/add/action.add.component';
import { ActionEditComponent } from './action/edit/action.edit.component';
import { ActionHelpComponent } from './action/help/action.help.component';
import { ActionListComponent } from './action/list/action.list.component';
import { ActionUsageComponent } from './action/usage/action.usage.component';
import { CdsctlComponent } from './cdsctl/cdsctl.component';
import { DownloadComponent } from './download/download.component';
import { GroupEditComponent } from './group/edit/group.edit.component';
import { GroupListComponent } from './group/list/group.list.component';
import { QueueComponent } from './queue/queue.component';
import { SettingsComponent } from './settings.component';
import { SettingsRouting } from './settings.routing';
import { UserEditComponent } from './user/edit/user.edit.component';
import { UserListComponent } from './user/list/user.list.component';
import { WorkerModelAddComponent } from './worker-model/add/worker-model.add.component';
import { WorkerModelEditComponent } from './worker-model/edit/worker-model.edit.component';
import { WorkerModelListComponent } from './worker-model/list/worker-model.list.component';
import { WorkflowTemplateAddComponent } from './workflow-template/add/workflow-template.add.component';
import { WorkflowTemplateEditComponent } from './workflow-template/edit/workflow-template.edit.component';
import { WorkflowTemplateEditorComponent } from './workflow-template/editor/workflow-template.editor.component';
import { WorkflowTemplateFormComponent } from './workflow-template/form/workflow-template.form.component';
import { WorkflowTemplateHelpComponent } from './workflow-template/help/workflow-template.help.component';
import { WorkflowTemplateListComponent } from './workflow-template/list/workflow-template.list.component';

@NgModule({
    declarations: [
        SettingsComponent,
        ActionAddComponent,
        ActionEditComponent,
        ActionListComponent,
        ActionUsageComponent,
        ActionHelpComponent,
        CdsctlComponent,
        DownloadComponent,
        GroupEditComponent,
        GroupListComponent,
        UserEditComponent,
        UserListComponent,
        WorkerModelAddComponent,
        WorkerModelEditComponent,
        WorkerModelListComponent,
        WorkflowTemplateEditorComponent,
        WorkflowTemplateFormComponent,
        WorkflowTemplateAddComponent,
        WorkflowTemplateEditComponent,
        WorkflowTemplateListComponent,
        WorkflowTemplateHelpComponent,
        QueueComponent
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
