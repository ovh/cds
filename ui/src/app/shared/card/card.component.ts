import { ChangeDetectionStrategy, Component, Input, OnDestroy, OnInit } from '@angular/core';

@Component({
    selector: 'app-card',
    templateUrl: './card.html',
    styleUrls: ['./card.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class CardComponent implements OnInit {

    @Input() image: string;
    @Input() title: string;
    @Input() description: string;

    ngOnInit(): void {
    }

}
