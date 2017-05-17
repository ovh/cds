import {Component, Input, OnInit} from '@angular/core';
import {WorkflowNode} from '../../../model/workflow.model';

@Component({
    selector: 'app-workflow-item',
    templateUrl: './workflow.node.html',
    styleUrls: ['./workflow.node.scss']
})
export class WorkflowNodeComponent implements OnInit {

    @Input() node: WorkflowNode;

    constructor() { }

    ngOnInit(): void {
    }
}
