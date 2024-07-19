import { ChangeDetectionStrategy, Component, Input } from "@angular/core";

@Component({
	selector: 'app-searchable',
	templateUrl: './searchable.html',
	styleUrls: ['./searchable.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class SearchableComponent {
	@Input() link: Array<string>;
	@Input() params: any;
	@Input() paramsHandling: string;
}