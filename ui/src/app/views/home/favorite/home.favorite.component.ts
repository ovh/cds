import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { NavbarProjectData } from 'app/model/navbar.model';


@Component({
    selector: 'app-home-favorite',
    templateUrl: './home.favorite.html',
    styleUrls: ['./home.favorite.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class HomeFavoriteComponent {

    @Input() bookmarks:  Array<NavbarProjectData>;
}
