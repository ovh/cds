import { ChangeDetectionStrategy, Component, Input } from '@angular/core';

@Component({
    selector: 'app-card',
    templateUrl: './card.html',
    styleUrls: ['./card.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class CardComponent {

    @Input() image: string;
    @Input() title: string;
    @Input() description: string;

    @Input() onlyTitle: boolean = false;
}
