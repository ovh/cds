import {Component, Input} from '@angular/core';

@Component({
    selector: 'app-zone-content',
    templateUrl: './content.html',
    styleUrls: ['./content.scss']
})
export class ZoneContentComponent {

    @Input() class: string;

    constructor() { }
}
