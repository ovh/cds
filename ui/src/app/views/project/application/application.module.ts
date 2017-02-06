import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../../shared/shared.module';
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

@NgModule({
    declarations: [
        ApplicationAdminComponent,
        ApplicationAddComponent,
        ApplicationRepositoryComponent,
        ApplicationPollerListComponent,
        ApplicationPollerFormComponent,
        ApplicationHookFormComponent,
        ApplicationHookListComponent,
        ApplicationShowComponent,
        ApplicationWorkflowComponent,
        ApplicationTreeWorkflowComponent,
        ApplicationWorkflowItemComponent,
        ApplicationTriggerComponent
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
