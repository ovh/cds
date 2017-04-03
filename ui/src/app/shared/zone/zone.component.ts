import {Component, Input} from '@angular/core';

@Component({
    selector: 'app-zone',
    templateUrl: './zone.html',
    styleUrls: ['./zone.scss']
})
export class ZoneComponent {

    @Input() header: string;
    @Input() headerClass: string;

    constructor() { }
}
