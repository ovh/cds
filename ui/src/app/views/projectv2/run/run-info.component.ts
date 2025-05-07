import { ChangeDetectionStrategy, Component, Input } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { WorkflowRunInfo } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";

@Component({
	selector: 'app-run-info',
	templateUrl: './run-info.html',
	styleUrls: ['./run-info.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunInfoComponent {
	@Input() info: Array<WorkflowRunInfo>;
}
