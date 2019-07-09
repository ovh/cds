import { ChangeDetectionStrategy, Component } from '@angular/core';

@Component({
    selector: 'app-scrollview',
    templateUrl: './scrollview.html',
    styleUrls: ['./scrollview.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ScrollviewComponent { }
