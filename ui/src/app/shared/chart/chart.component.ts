import {Component, Input} from '@angular/core';
import {GraphConfiguration} from '../../model/graph.model';

@Component({
    selector: 'app-chart',
    templateUrl: './chart.html',
    styleUrls: ['./chart.scss']
})
export class ChartComponentComponent {

    @Input() configuration: GraphConfiguration;
    view = [0, 200];
}
