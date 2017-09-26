import {AfterViewInit, Component, ElementRef, Input, ViewChild} from '@angular/core';
import {WorkflowNodeHook} from '../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
})
export class WorkflowNodeHookComponent implements AfterViewInit {

    @Input() hook: WorkflowNodeHook;
    @Input() readonly = false;

    @ViewChild('editHook')
    editHook: any;

    loading = false;

    constructor(private elementRef: ElementRef) {
    console.log('create hook compo');
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = 0;
    }

    openEditHookModal(): void {
        if (this.editHook) {
            this.editHook.show();
        }
    }
}
