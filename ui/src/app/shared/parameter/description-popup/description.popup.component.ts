import {Component, Input} from '@angular/core';
import { Parameter } from 'app/model/parameter.model';

@Component({
    selector: 'app-parameter-description',
    templateUrl: './parameter.description.html',
    styleUrls: ['./parameter.description.scss']
})
export class ParameterDescriptionComponent {

    @Input() parameter: Parameter;

    constructor() { }
}
