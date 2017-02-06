import {NgModule} from '@angular/core';
import {HomeComponent} from './home.component';
import { homeRouting } from './home.routing';
import {SharedModule} from '../../shared/shared.module';

@NgModule({
    declarations: [
        HomeComponent,
    ],
    imports: [
        SharedModule,
        homeRouting
    ],
    providers: [

    ]
})
export class HomeModule {
}
