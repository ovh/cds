import {NgModule} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {HomeBroadcastComponent} from './broadcast/home.broadcast.component';
import {HomeFavoriteComponent} from './favorite/home.favorite.component';
import {HomeComponent} from './home.component';
import { homeRouting } from './home.routing';
import {HomeTimelineFilterComponent} from './timeline/filter/home.timeline.filter.component';
import {HomeTimelineComponent} from './timeline/home.timeline.component';

@NgModule({
    declarations: [
        HomeComponent,
        HomeBroadcastComponent,
        HomeFavoriteComponent,
        HomeTimelineComponent,
        HomeTimelineFilterComponent
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
