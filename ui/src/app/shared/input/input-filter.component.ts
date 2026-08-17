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

/**
 * Serialization helpers for the raw filter text of app-input-filter.
 * A filter text is a space separated list of tokens, a token being either a `key:value` filter
 * or a free text search word. Free text words are carried by the `query` query param.
 */
export class FilterText {
	static readonly queryKey = 'query';

	static parse(text: string, opts?: { skipEmptyValues?: boolean }): { filters: { [key: string]: Array<string> }, query: Array<string> } {
		const filters: { [key: string]: Array<string> } = {};
		const query: Array<string> = [];
		(text ?? '').split(' ').forEach(token => {
			if (token === '') { return; }
			const splitted = token.split(':');
			if (splitted.length === 2) {
				if (opts?.skipEmptyValues && splitted[1] === '') { return; }
				const key = FilterText.decodeSpaces(splitted[0]);
				if (!filters[key]) { filters[key] = []; }
				filters[key].push(FilterText.decodeSpaces(splitted[1]));
			} else if (splitted.length === 1) {
				query.push(FilterText.decodeSpaces(token));
			}
		});
		return { filters, query };
	}

	/** Params sent to a search API: filters as arrays, free text words joined under the query key. */
	static toSearchParams(text: string): { [key: string]: any } {
		const { filters, query } = FilterText.parse(text);
		let params: { [key: string]: any } = { ...filters };
		if (query.length > 0) {
			params[FilterText.queryKey] = query.join(' ');
		}
		return params;
	}

	/** Router query params, dropping the filters left without a value. */
	static toQueryParams(text: string): { [key: string]: any } {
		const { filters, query } = FilterText.parse(text, { skipEmptyValues: true });
		let params: { [key: string]: any } = {};
		Object.keys(filters).forEach(key => {
			params[key] = filters[key].length === 1 ? filters[key][0] : filters[key];
		});
		if (query.length > 0) {
			params[FilterText.queryKey] = query.join(' ');
		}
		return params;
	}

	static fromQueryParams(values: { [key: string]: any }, ignoredKeys: Array<string> = []): string {
		const keys = Object.keys(values ?? {})
			.filter(key => ignoredKeys.indexOf(key) === -1)
			// Keep the free text search last, so that the key:value filters stay easy to spot.
			.sort((a, b) => (a === FilterText.queryKey ? 1 : 0) - (b === FilterText.queryKey ? 1 : 0));
		return keys.map(key => {
			const values_ = !Array.isArray(values[key]) ? [values[key]] : values[key];
			return values_.map(v => key === FilterText.queryKey ? `${v}` :
				`${FilterText.encodeSpaces(key)}:${FilterText.encodeSpaces(`${v}`)}`).join(' ');
		}).join(' ').trim();
	}

	private static encodeSpaces(v: string): string {
		return v.split(' ').join(InputFilterComponent.spaceAlternative);
	}

	private static decodeSpaces(v: string): string {
		return v.split(InputFilterComponent.spaceAlternative).join(' ');
	}
}

@Component({
	standalone: false,
	selector: 'app-input-filter',
	templateUrl: './input-filter.html',
	changeDetection: ChangeDetectionStrategy.OnPush,
	encapsulation: ViewEncapsulation.None
})
@AutoUnsubscribe()
export class InputFilterComponent<T> implements AfterViewInit, AfterViewChecked, OnDestroy {
	static readonly spaceAlternative = '\u00A0';

	@ViewChild('filterInput') filterInput: ElementRef;
	@ViewChild('filterInputDirective') filterInputDirective: NzAutocompleteTriggerDirective;
	@ViewChildren(NzAutocompleteOptionComponent) fromDataSourceOptions: QueryList<NzAutocompleteOptionComponent>;

	@Input() placeholder: string = '';
	@Input() filterText: string = '';
	@Input() filters: Array<Filter> = [];
	@Input() suggestions: Array<Suggestion<T>> = [];
	@Input() suggestionTemplate: TemplateRef<unknown> | undefined;
	@Input() autofocus: boolean = false;
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
					this.filterInputDirective.closePanel();
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
		const setTriggerValue = (this.filterInputDirective as any).setTriggerValue.bind(this.filterInputDirective);
		(this.filterInputDirective as any).doBackfill = (): void => {
			if (this.filterInputDirective.nzAutocomplete.activeItem.getLabel().indexOf('jump:') === 0) {
				setTriggerValue(this.filterText);
				return;
			}
			doBackfill();
		}

		if (this.autofocus) {
			// Focusing opens the autocomplete panel; close it back so the page stays
			// readable until the user actually types or clicks.
			setTimeout(() => {
				this.filterInput.nativeElement.focus({ preventScroll: true });
				this.filterInputDirective.closePanel();
			});
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
		this.textFilters = (originalText ?? '').split(' ');
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
			this.selectedFilter = Object.assign({}, this.filters.find(f => f.key === splitted[0].replace(InputFilterComponent.spaceAlternative, ' ')));
			if (this.selectedFilter) {
				this.selectedFilter.options = (this.selectedFilter.options ?? []).filter(o => splitted[1] === '' || o.toLowerCase().indexOf(splitted[1].replace(InputFilterComponent.spaceAlternative, ' ').toLowerCase()) !== -1);
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
		textFilters[this.cursorTextFilterPosition] = filter.key.replace(' ', InputFilterComponent.spaceAlternative) + ':' + (option ? option.replace(' ', InputFilterComponent.spaceAlternative) : '');
		return textFilters.join(' ');
	}
}