import { Pipe, PipeTransform } from '@angular/core';
import { V2WorkflowRunJobStatus, V2WorkflowRunJobStatusIsTerminated } from './v2.workflow.run.model';

@Pipe({
    name: 'isJobTerminated',
    standalone: false
})
export class IsJobTerminatedPipe implements PipeTransform {
	transform(value: V2WorkflowRunJobStatus): boolean {
		return V2WorkflowRunJobStatusIsTerminated(value);
	}
}
