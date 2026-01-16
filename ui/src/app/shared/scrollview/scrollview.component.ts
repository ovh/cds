import { ChangeDetectionStrategy, Component } from '@angular/core';

@Component({
    standalone: false,
    selector: 'app-scrollview',
    templateUrl: './scrollview.html',
    styleUrls: ['./scrollview.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ScrollviewComponent { }
