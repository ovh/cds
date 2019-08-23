import { ChangeDetectionStrategy, Component } from '@angular/core';

@Component({
    selector: 'app-admin',
    templateUrl: './admin.html',
    styleUrls: ['./admin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class AdminComponent {
    constructor() { }
}
