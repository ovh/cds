import {Component, Input} from '@angular/core';
import {NavbarProjectData} from '../../../model/navbar.model';

@Component({
    selector: 'app-home-favorite',
    templateUrl: './home.favorite.html',
    styleUrls: ['./home.favorite.scss']
})
export class HomeFavoriteComponent {

    @Input() bookmarks:  Array<NavbarProjectData>;
}
