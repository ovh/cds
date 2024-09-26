import { ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, Input, OnChanges, OnInit, SimpleChanges, ViewChild } from "@angular/core";
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from "@angular/forms";
import { Store } from "@ngxs/store";
import { Branch, Tag } from "app/model/repositories.model";
import { OnChangeType, OnTouchedType } from "ng-zorro-antd/core/types";
import { AutoUnsubscribe } from "../decorator/autoUnsubscribe";
import { Subscription } from "rxjs";
import { PreferencesState } from "app/store/preferences.state";
import { NzSelectComponent } from "ng-zorro-antd/select";

@Component({
	selector: 'app-repository-ref-select',
	templateUrl: './repository-ref-select.html',
	styleUrls: ['./repository-ref-select.scss'],
	providers: [
		{
			provide: NG_VALUE_ACCESSOR,
			useExisting: forwardRef(() => RepositoryRefSelectComponent),
			multi: true
		}
	],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RepositoryRefSelectComponent implements OnInit, OnChanges, ControlValueAccessor {
	@ViewChild('select') select: NzSelectComponent;

	@Input() size: string = '';
	@Input() branches: Array<Branch>;
	@Input() tags: Array<Tag>;

	filteredBranches: Array<Branch>;
	filteredTags: Array<Tag>;

	onChange: OnChangeType = () => { };
	onTouched: OnTouchedType = () => { };
	selectedRef: string = '';
	themeSubscription: Subscription;
	darkActive: boolean;
	filter: string;
	disabled: boolean;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) { }

	ngOnInit(): void {
		this.themeSubscription = this._store.select(PreferencesState.theme).subscribe(t => {
			this.darkActive = t === 'night';
			this._cd.markForCheck();
		});
	}

	ngOnChanges(changes: SimpleChanges): void {
		this.filterRef();
	}

	registerOnChange(fn: OnChangeType): void { this.onChange = fn; }

	registerOnTouched(fn: OnTouchedType): void { this.onTouched = fn; }

	setDisabledState?(isDisabled: boolean): void {
		this.disabled = isDisabled;
		this._cd.markForCheck();
	}

	writeValue(v: string): void {
		this.selectedRef = v;
		this._cd.markForCheck();
	}

	clickOption(v: string): void {
		this.selectedRef = v;
		this._cd.markForCheck();
		this.onChange(v);
		this.select.setOpenState(false);
	}

	onFilterRefChange(v: any): void {
		this.filter = (v.target.value ?? '').toLowerCase();
		this.filterRef();
	}

	filterRef(): void {
		this.filteredBranches = (this.branches ?? []).filter(b => !this.filter || b.display_id.toLowerCase().indexOf(this.filter) !== -1);
		this.filteredTags = (this.tags ?? []).filter(t => !this.filter || t.tag.toLowerCase().indexOf(this.filter) !== -1);
		this._cd.markForCheck();
	}
}