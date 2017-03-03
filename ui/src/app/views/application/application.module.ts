import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {applicationRouting} from './application.routing';
import {ApplicationShowComponent} from './show/application.component';
import {ApplicationAddComponent} from './add/application.add.component';
import {ApplicationAdminComponent} from './show/admin/application.admin.component';
import {ApplicationRepositoryComponent} from './show/admin/repository/application.repo.component';
import {ApplicationPollerListComponent} from './show/admin/poller/list/application.poller.list.component';
import {ApplicationPollerFormComponent} from './show/admin/poller/form/application.poller.form.component';
import {ApplicationHookFormComponent} from './show/admin/hook/form/application.hook.form.component';
import {ApplicationHookListComponent} from './show/admin/hook/list/application.hook.list.component';
import {ApplicationWorkflowComponent} from './show/workflow/application.workflow.component';
import {ApplicationTreeWorkflowComponent} from './show/workflow/tree/application.tree.workflow.component';
import {ApplicationWorkflowItemComponent} from './show/workflow/tree/item/application.workflow.item.component';
import {ApplicationTriggerComponent} from './show/workflow/trigger/trigger.component';
import {ApplicationNotificationListComponent} from './show/notifications/list/notification.list.component';
import {ApplicationNotificationFormModalComponent} from './show/notifications/form/notification.form.component';
import {ApplicationPipelineLinkComponent} from './show/workflow/pipeline/pipeline.link.component';
import {ApplicationSchedulerItemComponent} from './show/scheduler/item/scheduler.item.component';
import {ApplicationSchedulerFormComponent} from './show/scheduler/form/scheduler.form.component';

@NgModule({
    declarations: [
        ApplicationAdminComponent,
        ApplicationAddComponent,
        ApplicationHookFormComponent,
        ApplicationHookListComponent,
        ApplicationNotificationFormModalComponent,
        ApplicationNotificationListComponent,
        ApplicationPipelineLinkComponent,
        ApplicationPollerFormComponent,
        ApplicationPollerListComponent,
        ApplicationRepositoryComponent,
        ApplicationSchedulerItemComponent,
        ApplicationSchedulerFormComponent,
        ApplicationShowComponent,
        ApplicationTreeWorkflowComponent,
        ApplicationTriggerComponent,
        ApplicationWorkflowComponent,
        ApplicationWorkflowItemComponent
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
