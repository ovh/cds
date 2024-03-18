import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, SimpleChanges } from "@angular/core";
import moment from 'moment';

@Component({
	selector: 'app-date-from-now',
	templateUrl: './date-from-now.html',
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class DateFromNowComponent implements OnChanges {
	@Input() value: string;

	fromNow: string = '';

	constructor(
		private _cd: ChangeDetectorRef
	) { }

	ngOnChanges(changes: SimpleChanges): void {
		try {
			const date = moment(this.value);
			this.fromNow = date.fromNow();
		} catch (e) { }
		this._cd.markForCheck();
	}

}