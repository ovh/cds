import { ChangeDetectionStrategy, Component, Input } from "@angular/core";

/**
 * Displays a CDS entity as its name first, its ref and its vcs/repository scope following it in a
 * lighter style. Used wherever a workflow is named, so that they all read alike.
 *
 * The ref is only meaningful when naming a workflow definition. Next to a run it would be read as
 * the ref the run checked out, which it is not, so runs are named without it.
 *
 * Give a link to make the name alone a link, or wrap the whole component in one to make everything
 * clickable.
 */
@Component({
	standalone: false,
	selector: 'app-workflow-name',
	templateUrl: './workflow-name.html',
	styleUrls: ['./workflow-name.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowNameComponent {
	@Input() name: string;
	@Input() scope: string;
	/** Rendered right after the name, for a run number for example. */
	@Input() suffix: string;
	@Input() link: Array<any>;
	@Input() linkQueryParams: { [key: string]: any };
	@Input() linkTitle: string;

	@Input() set ref(value: string) {
		this.displayRef = WorkflowNameComponent.shortRef(value);
	}

	displayRef: string;

	/** Refs are displayed without their `refs/heads/` or `refs/tags/` prefix. */
	static shortRef(ref: string): string {
		return (ref ?? '').replace(/^refs\/(heads|tags)\//, '');
	}
}
