import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { SharedModule } from '../../shared/shared.module';
import { EnvironmentAddComponent } from './add/environment.add.component';
import { environmentRouting } from './environment.routing';
// import { EnvironmentShowComponent } from './show/environment.show.component';
// import { EnvironmentKeysComponent } from './show/keys/environment.keys.component';


@NgModule({
    declarations: [
        EnvironmentAddComponent,
        // EnvironmentShowComponent,
        // EnvironmentKeysComponent,
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
