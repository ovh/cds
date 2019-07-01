import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { SharedModule } from 'app/shared/shared.module';
import { EnvironmentAddComponent } from './add/environment.add.component';
import { environmentRouting } from './environment.routing';
import { EnvironmentAdvancedComponent } from './show/advanced/environment.advanced.component';
import { EnvironmentShowComponent } from './show/environment.show.component';
import { EnvironmentKeysComponent } from './show/keys/environment.keys.component';


@NgModule({
    declarations: [
        EnvironmentAddComponent,
        EnvironmentShowComponent,
        EnvironmentKeysComponent,
        EnvironmentAdvancedComponent,
    ],
    imports: [
        SharedModule,
        environmentRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class EnvironmentModule {
}
