import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {applicationRouting} from './application.routing';
import {ApplicationShowComponent} from './show/application.component';
import {ApplicationAddComponent} from './add/application.add.component';
import {ApplicationAdminComponent} from './show/admin/application.admin.component';
import {ApplicationRepositoryComponent} from './show/admin/repository/application.repo.component';
import {ApplicationWorkflowComponent} from './show/workflow/application.workflow.component';
import {ApplicationTreeWorkflowComponent} from './show/workflow/tree/application.tree.workflow.component';
import {ApplicationWorkflowItemComponent} from './show/workflow/tree/item/application.workflow.item.component';
import {ApplicationTriggerComponent} from './show/workflow/trigger/trigger.component';
import {ApplicationNotificationListComponent} from './show/notifications/list/notification.list.component';
import {ApplicationNotificationFormModalComponent} from './show/notifications/form/notification.form.component';
import {ApplicationPipelineLinkComponent} from './show/workflow/pipeline/link/pipeline.link.component';
import {ApplicationSchedulerItemComponent} from './show/scheduler/item/scheduler.item.component';
import {ApplicationSchedulerFormComponent} from './show/scheduler/form/scheduler.form.component';
import {ApplicationHookItemComponent} from './show/hook/item/hook.item.component';
import {ApplicationPollerItemComponent} from './show/poller/item/poller.item.component';
import {ApplicationHookItemFormComponent} from './show/hook/edit/item.form.component';
import {ApplicationPollerFormComponent} from './show/poller/edit/poller.edit.component';
import {ApplicationPipelineDetachComponent} from './show/workflow/pipeline/detach/pipeline.detach.component';
import {ApplicationKeysComponent} from './show/keys/appplication.keys.component';

@NgModule({
    declarations: [
        ApplicationAdminComponent,
        ApplicationAddComponent,
        ApplicationHookItemComponent,
        ApplicationHookItemFormComponent,
        ApplicationNotificationFormModalComponent,
        ApplicationNotificationListComponent,
        ApplicationPipelineDetachComponent,
        ApplicationPipelineLinkComponent,
        ApplicationPollerItemComponent,
        ApplicationPollerFormComponent,
        ApplicationRepositoryComponent,
        ApplicationSchedulerItemComponent,
        ApplicationSchedulerFormComponent,
        ApplicationShowComponent,
        ApplicationTreeWorkflowComponent,
        ApplicationTriggerComponent,
        ApplicationWorkflowComponent,
        ApplicationWorkflowItemComponent,
        ApplicationKeysComponent
    ],
    imports: [
        SharedModule,
        applicationRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ApplicationModule {
}
