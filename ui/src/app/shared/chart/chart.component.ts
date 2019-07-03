import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { GraphConfiguration } from 'app/model/graph.model';

@Component({
    selector: 'app-chart',
    templateUrl: './chart.html',
    styleUrls: ['./chart.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ChartComponentComponent {

    @Input() configuration: GraphConfiguration;
}
