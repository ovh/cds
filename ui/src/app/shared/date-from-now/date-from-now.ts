import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnDestroy, SimpleChanges } from "@angular/core";
import moment from 'moment';
import { AutoUnsubscribe } from "../decorator/autoUnsubscribe";
import { interval, Subscription } from "rxjs";

@Component({
	selector: 'app-date-from-now',
	templateUrl: './date-from-now.html',
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class DateFromNowComponent implements OnChanges, OnDestroy {
	@Input() value: string;

	refresh: Subscription;
	fromNow: string = '';
	date: any;

	constructor(
		private _cd: ChangeDetectorRef
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnChanges(changes: SimpleChanges): void {
		try {
			this.date = moment(this.value);
		} catch (e) { }
		if (!this.refresh && this.date) {
			this.refresh = interval(30000).subscribe(() => { this.computeFromNow(); });
		}
		this.computeFromNow();
	}

	computeFromNow(): void {
		if (!this.date) { return; }
		this.fromNow = this.date.fromNow();
		this._cd.markForCheck();
	}

}