import {NgModule} from '@angular/core';
import {SharedModule} from '../../shared/shared.module';
import {FavoriteComponent} from './favorite.component';
import { favoriteRouting } from './favorite.routing';

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
