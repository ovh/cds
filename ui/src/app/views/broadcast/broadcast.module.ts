import {NgModule} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import { broadcastRouting } from './broadcast.routing';
import {BroadcastDetailsComponent} from './details/broadcast.details.component';
import {BroadcastListComponent} from './list/broadcast.list.component';

@NgModule({
    declarations: [
        BroadcastListComponent,
        BroadcastDetailsComponent,
    ],
    imports: [
        SharedModule,
        broadcastRouting
    ],
    providers: [

    ]
})
export class BroadcastModule {
}
