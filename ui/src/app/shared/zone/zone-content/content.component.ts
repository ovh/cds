import { ChangeDetectionStrategy, Component, Input } from '@angular/core';

@Component({
    selector: 'app-zone-content',
    templateUrl: './content.html',
    styleUrls: ['./content.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ZoneContentComponent {

    @Input() class: string;

    constructor() { }
}
