import { Component, Input, OnInit } from '@angular/core';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-project-pipelines',
    templateUrl: './pipeline.list.html',
    styleUrls: ['./pipeline.list.scss']
})
export class ProjectPipelinesComponent {

    @Input() project: Project;

    constructor() {

    }
}
