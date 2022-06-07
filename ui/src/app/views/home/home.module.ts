import { NgModule } from '@angular/core';
import { SharedModule } from '../../shared/shared.module';
import { HomeFavoriteComponent } from './favorite/home.favorite.component';
import { HomeFilterComponent } from './filter/home.filter.component';
import { HomeHeatmapComponent } from './heatmap/home.heatmap.component';
import { ToolbarComponent } from './heatmap/toolbar/toolbar.component';
import { HomeComponent } from './home.component';
import { homeRouting } from './home.routing';
import { HomeTimelineComponent } from './timeline/home.timeline.component';


@NgModule({
    declarations: [
        HomeComponent,
        HomeFavoriteComponent,
        HomeFilterComponent,
        ToolbarComponent,
        HomeHeatmapComponent,
        HomeTimelineComponent,
    ],
    imports: [
        SharedModule,
        homeRouting
    ]
})
export class HomeModule {
}
