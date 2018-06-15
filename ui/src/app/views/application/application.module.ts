import {CUSTOM_ELEMENTS_SCHEMA, NgModule} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {ApplicationAddComponent} from './add/application.add.component';
import {applicationRouting} from './application.routing';
import {ApplicationAdminComponent} from './show/admin/application.admin.component';
import {ApplicationDeploymentComponent} from './show/admin/deployment/application.deployment.component';
import {ApplicationRepositoryComponent} from './show/admin/repository/application.repo.component';
import {ApplicationShowComponent} from './show/application.component';
import {ApplicationHookItemFormComponent} from './show/hook/edit/item.form.component';
import {ApplicationHookItemComponent} from './show/hook/item/hook.item.component';
import {ApplicationKeysComponent} from './show/keys/appplication.keys.component';
import {ApplicationNotificationFormModalComponent} from './show/notifications/form/notification.form.component';
import {ApplicationNotificationListComponent} from './show/notifications/list/notification.list.component';
import {ApplicationPollerFormComponent} from './show/poller/edit/poller.edit.component';
import {ApplicationPollerItemComponent} from './show/poller/item/poller.item.component';
import {ApplicationSchedulerFormComponent} from './show/scheduler/form/scheduler.form.component';
import {ApplicationSchedulerItemComponent} from './show/scheduler/item/scheduler.item.component';
import {ApplicationWorkflowComponent} from './show/workflow/application.workflow.component';
import {ApplicationPipelineDetachComponent} from './show/workflow/pipeline/detach/pipeline.detach.component';
import {ApplicationPipelineLinkComponent} from './show/workflow/pipeline/link/pipeline.link.component';
import {ApplicationTreeWorkflowComponent} from './show/workflow/tree/application.tree.workflow.component';
import {ApplicationWorkflowItemComponent} from './show/workflow/tree/item/application.workflow.item.component';
import {ApplicationTriggerComponent} from './show/workflow/trigger/trigger.component';

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
        ApplicationDeploymentComponent,
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
