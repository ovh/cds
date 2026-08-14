import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from "@angular/core";
import { Project } from "app/model/project.model";
import { VariableSet } from "app/model/variablesets.model";
import { V2ProjectService } from "app/service/projectv2/project.service";
import { ErrorUtils } from "app/shared/error.utils";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom } from "rxjs";

@Component({
    standalone: false,
    selector: 'app-project-variable-sets',
    templateUrl: './variablesets.html',
    styleUrls: ['./variablesets.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectVariableSetsComponent implements OnInit {
    @Input() project: Project;

    loading = { list: false, action: false };
    newVariableSetName: string;
    selectedVariableSet: VariableSet;
    variableSets: Array<VariableSet> = [];
    filteredVariableSets: Array<VariableSet> = [];
    filter: string = '';
    createModalVisible: boolean = false;

    constructor(
        private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _v2ProjectService: V2ProjectService
    ) { }

    ngOnInit(): void {
        this.load();
    }

    async load() {
        this.loading.list = true;
        this._cd.markForCheck();
        try {
            this.variableSets = await lastValueFrom(this._v2ProjectService.getVariableSets(this.project.key));
            this.applyFilter();
        } catch (e) {
            this._messageService.error(`Unable to load variables sets: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.list = false;
        this._cd.markForCheck();
    }

    async createVariableSet() {
        if (!this.newVariableSetName) {
            return;
        }
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            const set = await lastValueFrom(this._v2ProjectService.createVariableSet(this.project.key, <VariableSet>{ name: this.newVariableSetName }))
            this.variableSets = this.variableSets.concat(set)
            this.variableSets.sort((v1, v2) => v1.name < v2.name ? -1 : 1);
            this.applyFilter();
            this.closeCreateModal();
        } catch (e) {
            this._messageService.error(`Unable to set variables set: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    async deleteVariableSet(v: VariableSet) {
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._v2ProjectService.deleteVariableSet(this.project.key, v.name))
            this.variableSets = this.variableSets.filter(s => s.name !== v.name);
            this.applyFilter();
            if (this.selectedVariableSet?.name === v.name) {
                this.unselectVariableSet();
            }
        } catch (e) {
            this._messageService.error(`Unable to delete variables set: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    updateFilter(value: string): void {
        this.filter = value;
        this.applyFilter();
        this._cd.markForCheck();
    }

    applyFilter(): void {
        const search = (this.filter ?? '').trim().toLowerCase();
        this.filteredVariableSets = search === '' ? this.variableSets :
            this.variableSets.filter(v => v.name.toLowerCase().indexOf(search) !== -1);
    }

    openCreateModal(): void {
        this.newVariableSetName = '';
        this.createModalVisible = true;
        this._cd.markForCheck();
    }

    closeCreateModal(): void {
        this.createModalVisible = false;
        this._cd.markForCheck();
    }

    selectVariableSet(v: VariableSet): void {
        this.selectedVariableSet = v;
        this._cd.markForCheck();
    }

    unselectVariableSet(): void {
        delete this.selectedVariableSet;
        this._cd.markForCheck();
    }
}
