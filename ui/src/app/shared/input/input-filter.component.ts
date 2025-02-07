import { AfterViewChecked, AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, EventEmitter, Input, OnDestroy, Output, QueryList, TemplateRef, ViewChild, ViewChildren, ViewEncapsulation } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { NzAutocompleteOptionComponent, NzAutocompleteTriggerDirective } from "ng-zorro-antd/auto-complete";

export class Filter {
	key: string;
	options: Array<string>;
	example: string;
}

export class FilterValue {
	key: string;
	value: string;
}

export class Suggestion<T> {
	key: string;
	label: string
	data: T;
}

@Component({
	selector: 'app-input-filter',
	templateUrl: './input-filter.html',
	changeDetection: ChangeDetectionStrategy.OnPush,
	encapsulation: ViewEncapsulation.None
})
@AutoUnsubscribe()
export class InputFilterComponent<T> implements AfterViewInit, AfterViewChecked, OnDestroy {
	@ViewChild('filterInput') filterInput: ElementRef;
	@ViewChild('filterInputDirective') filterInputDirective: NzAutocompleteTriggerDirective;
	@ViewChildren(NzAutocompleteOptionComponent) fromDataSourceOptions: QueryList<NzAutocompleteOptionComponent>;

	@Input() placeholder: string = '';
	@Input() filterText: string = '';
	@Input() filters: Array<Filter> = [];
	@Input() suggestions: Array<Suggestion<T>> = [];
	@Input() suggestionTemplate: TemplateRef<unknown> | undefined; cds
	@Output() changeFilter: EventEmitter<string> = new EventEmitter();
	@Output() selectSuggestion: EventEmitter<T> = new EventEmitter();
	@Output() submit: EventEmitter<void> = new EventEmitter();

	textFilters = [];
	cursorTextFilterPosition: number = 0;
	selectedFilter: Filter = null;
	availableFilters: Array<Filter> = [];

	constructor(
		private _cd: ChangeDetectorRef
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnChange(): void {
		this.computeAvailableFilters(this.filterText);
	}

	ngAfterViewInit(): void {
		const callback = this.filterInputDirective.handleKeydown.bind(this.filterInputDirective);
		this.filterInputDirective.handleKeydown = (event: KeyboardEvent): void => {
			if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
				this.computeAvailableFilters(this.filterText);
			}
			if ((event.key === 'ArrowLeft' || event.key === 'ArrowRight' || event.key === 'ArrowDown') && !this.filterInputDirective.panelOpen) {
				this.filterInputDirective.openPanel();
				return;
			}
			if (event.key === 'Enter') {
				if (this.filterInputDirective.activeOption && this.filterInputDirective.activeOption.getLabel().indexOf('jump:') === 0) {
					this.selectSuggestion.emit(this.filterInputDirective.activeOption.nzValue);
					return;
				}
				if (this.filterInputDirective.activeOption && this.filterInputDirective.activeOption.nzValue !== this.filterText) {
					if (this.filterInputDirective.activeOption.nzValue.endsWith(':')) {
						event.preventDefault();
					}
					this.onFilterTextChange(this.filterInputDirective.activeOption.nzValue);
					return;
				} else if (this.filterInputDirective.activeOption) {
					this.changeFilter.emit(this.filterText);
				}
			}
			if (event.key === 'Escape') {
				this.filterInputDirective.closePanel();
				return;
			}
			callback(event);
		};

		const doBackfill = (this.filterInputDirective as any).doBackfill.bind(this.filterInputDirective);
		(this.filterInputDirective as any).doBackfill = (): void => {
			if (this.filterInputDirective.nzAutocomplete.activeItem.getLabel().indexOf('jump:') === 0) {
				return;
			}
			doBackfill();
		}
	}

	ngAfterViewChecked(): void {
		this.fromDataSourceOptions.forEach(o => {
			o.selectViaInteraction = () => {
				this.onFilterTextChange(o.nzValue);
				if (!o.nzValue.endsWith(':')) {
					this.submit.emit();
					this.filterInputDirective.closePanel();
				}
			}
		});
	}

	onFilterTextChange(originalText: string): void {
		this.computeAvailableFilters(originalText);
		this.filterText = originalText;
		this.changeFilter.emit(this.filterText);
		this._cd.markForCheck();
	}

	computeAvailableFilters(originalText: string): void {
		// Get and adjust cursor position
		const originalCursorPosition = this.filterInput.nativeElement.selectionStart;
		this.textFilters = originalText.split(' ');
		// Retrieve the active filter in the text
		this.cursorTextFilterPosition = 0;
		let count = 0;
		this.textFilters.forEach((filter, idx) => {
			if (idx > 0) { count++ }; // Add +1 that match the space
			if (count <= originalCursorPosition && originalCursorPosition <= count + filter.length) {
				this.cursorTextFilterPosition = idx;
			}
			count += filter.length;
		});

		const splitted = this.textFilters[this.cursorTextFilterPosition].split(':');
		if (splitted.length === 2) {
			// Search for existing filter key to show options
			this.selectedFilter = Object.assign({}, this.filters.find(f => f.key === splitted[0]));
			if (this.selectedFilter) {
				this.selectedFilter.options = (this.selectedFilter.options ?? []).filter(o => splitted[1] === '' || o.toLowerCase().indexOf(splitted[1].toLowerCase()) !== -1);
			}
			this.availableFilters = [];
		} else {
			this.availableFilters = [].concat(this.filters);
			this.selectedFilter = null;
		}
	}

	onClickInput(): void {
		this.computeAvailableFilters(this.filterText);
		if (!this.filterInputDirective.panelOpen) {
			this.filterInputDirective.openPanel();
		}
	}

	computeFilterValue(filter: Filter, option?: string): string {
		const textFilters = [].concat(this.textFilters);
		textFilters[this.cursorTextFilterPosition] = filter.key + ':' + (option ? encodeURI(option) : '');
		return textFilters.join(' ');
	}
}