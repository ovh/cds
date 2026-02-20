import { ChangeDetectionStrategy, Component, Input } from "@angular/core";
import { QueryParamsHandling } from "@angular/router";

@Component({
    standalone: false,
	selector: 'app-searchable',
	templateUrl: './searchable.html',
	styleUrls: ['./searchable.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class SearchableComponent {
	@Input() link: Array<string>;
	@Input() params: any;
	@Input() paramsHandling: QueryParamsHandling = '';
	@Input() fixed: boolean;
	@Input() title: string;
}
