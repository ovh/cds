import {CUSTOM_ELEMENTS_SCHEMA, NgModule, NO_ERRORS_SCHEMA} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {WarningShowComponent} from './show/warning.show.component';
import {warningRouting} from './warnings.routing';
import {WarningBreadCrumbComponent} from './breadcrumb/warning.breadcrumb.component';

@NgModule({
    declarations: [
        WarningBreadCrumbComponent,
        WarningShowComponent
    ],
    imports: [
        SharedModule,
        warningRouting,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA,
        NO_ERRORS_SCHEMA
    ]
})
export class WarningsModule {
}
