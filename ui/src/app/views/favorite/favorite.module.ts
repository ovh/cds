import {NgModule} from '@angular/core';
import {FavoriteComponent} from './favorite.component';
import { favoriteRouting } from './favorite.routing';
import {SharedModule} from '../../shared/shared.module';

@NgModule({
    declarations: [
        FavoriteComponent,
    ],
    imports: [
        SharedModule,
        favoriteRouting
    ],
    providers: [

    ]
})
export class FavoriteModule {
}
