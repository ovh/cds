import {Component, Input} from '@angular/core';
import {Subscription} from 'rxjs/Subscription';
import {Project} from '../../../model/project.model';
import {WorkflowEventStore} from '../../../service/workflow/workflow.event.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-worflow-breadcrumb',
    templateUrl: './breadcrumb.html'
})
@AutoUnsubscribe()
export class WorkflowBreadCrumbComponent {

    @Input() project: Project;
    @Input() workflowName: string;
    run: number;

    runSub: Subscription;

    constructor(private _workflowEventStore: WorkflowEventStore) {
        this.runSub = this._workflowEventStore.selectedRun().subscribe(wr => {
            if (wr) {
                this.run = wr.num;
            } else {
                delete this.run;
            }
        });

    }
}
