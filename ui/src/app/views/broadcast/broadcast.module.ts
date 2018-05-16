import {NgModule} from '@angular/core';
import {BroadcastListComponent} from './list/broadcast.list.component';
import {BroadcastDetailsComponent} from './details/broadcast.details.component';
import { broadcastRouting } from './broadcast.routing';
import {SharedModule} from '../../shared/shared.module';

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
