import { Component } from '@angular/core';
import { WorkflowHookTask } from '../../../model/workflow.hook.model';
import { HookService } from '../../../service/services.module';

@Component({
    selector: 'app-hooks-tasks',
    templateUrl: './hooks-tasks.html',
    styleUrls: ['./hooks-tasks.scss']
})
export class HooksTasksComponent {
    loading = false;
    tasks: Array<WorkflowHookTask>;

    constructor(private _hookService: HookService) {
        this.loading = true;
        this._hookService.getAdminTasks('')
            .subscribe(ts => {
                this.tasks = ts;
                this.loading = false;
            });
    }
}
