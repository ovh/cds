import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { SharedModule } from '../../shared/shared.module';
import { ApplicationAddComponent } from './add/application.add.component';
import { applicationRouting } from './application.routing';
import { ApplicationAdminComponent } from './show/admin/application.admin.component';
import { ApplicationDeploymentComponent } from './show/admin/deployment/application.deployment.component';
import { ApplicationRepositoryComponent } from './show/admin/repository/application.repo.component';
import { ApplicationShowComponent } from './show/application.component';
import { ApplicationHomeComponent } from './show/home/application.home.component';
import { ApplicationKeysComponent } from './show/keys/application.keys.component';


@NgModule({
    declarations: [
        ApplicationAdminComponent,
        ApplicationAddComponent,
        ApplicationHomeComponent,
        ApplicationRepositoryComponent,
        ApplicationDeploymentComponent,
        ApplicationShowComponent,
        ApplicationKeysComponent,
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
