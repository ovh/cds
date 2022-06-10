import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input, OnChanges,
    OnInit,
    Output, SimpleChanges
} from '@angular/core';
import { Requirement } from 'app/model/requirement.model';
import { WorkerModel } from 'app/model/worker-model.model';

@Component({
    selector: 'app-requirements-value',
    templateUrl: './requirements.value.html',
    styleUrls: ['./requirements.value.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RequirementsValueComponent implements OnInit, OnChanges {

    @Input() requirement: Requirement
    @Output() requirementChange = new EventEmitter<Requirement>();

    @Input() edit;
    @Input() suggest: Array<string> = [];
    aggregatedSuggestions: Array<string> = [];

    filteredSuggest: Array<string> = [];

    @Input() suggestWorkerModels: Array<string> = [];
    @Input() workerModels: Map<string, WorkerModel> = new Map();
    @Input() suggestArchOs: string[] = [];

    constructor(private _cd: ChangeDetectorRef) {
    }

    ngOnInit(): void {
        this.initFilter(this.requirement);
        this._cd.markForCheck();
    }

    ngOnChanges(changes: SimpleChanges) {
        this.initFilter(this.requirement);
        this._cd.markForCheck();
    }

    initFilter(r: Requirement): void {
        this.requirement = r;
        this.aggregatedSuggestions = new Array<string>();
        if (r.type === 'os-architecture') {
            this.aggregatedSuggestions.push(...this.suggestArchOs);
        } else {
            if (r.type === 'model') {
                this.aggregatedSuggestions.push(...this.suggestWorkerModels);
            }
            this.aggregatedSuggestions.push(...this.suggest);
        }
        this.filteredSuggest = Object.assign([], this.aggregatedSuggestions);
    }

    change(r: Requirement): void {
        this.initFilter(r);
        this.filterSuggestion(r.value)
        this.requirementChange.next(r);
        this._cd.markForCheck();
    }

    filterSuggestion(value: string): void{
        this.filteredSuggest = this.aggregatedSuggestions.filter(s => s.indexOf(value) !== -1);
    }

    selectChange(e: any): void {
        console.log(this.requirement.value, e);
    }
}
