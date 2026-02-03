import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, SimpleChanges } from "@angular/core";
import { Project } from "app/model/project.model";
import { VariableSet, VariableSetItem } from "app/model/variablesets.model";
import { V2ProjectService } from "app/service/projectv2/project.service";
import { ErrorUtils } from "app/shared/error.utils";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom } from "rxjs";

@Component({
    standalone: false,
    selector: 'app-project-variable-set-items',
    templateUrl: './variableset.item.html',
    styleUrls: ['./variableset.item.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectVariableSetItemsComponent implements OnChanges {
    @Input() project: Project;
    @Input() variableSet: VariableSet

    items: VariableSetItem[];
    loading: boolean;
    itemFormLoading: boolean;
    newItem: VariableSetItem;
    errorItemName = false;
    errorItemValue = false;
    varsetItemPattern = new RegExp("^[a-zA-Z0-9_-]{1,}$")

    constructor(
        private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _v2ProjectService: V2ProjectService
    ) {
        this.newItem = new VariableSetItem();
    }

    ngOnChanges(changes: SimpleChanges): void {
        if (!this.variableSet || !this.project) {
            return
        }
        this.loadVariableSet();
    }

    async loadVariableSet() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            const res = await lastValueFrom(this._v2ProjectService.getVariableSet(this.project.key, this.variableSet.name));
            this.items = res.items
        } catch (e) {
            this._messageService.error(`Unable to load variable set: ${ErrorUtils.print(e)}`);
        }

        this.newItem = new VariableSetItem();
        this.loading = false;
        this._cd.markForCheck();
    }

    async createVariableSetItem() {
        if (!this.varsetItemPattern.test(this.newItem.name)) {
            this.errorItemName = true;
            this._cd.markForCheck();
            return;
        }
        this.errorItemName = false;

        if (this.newItem.value === '') {
            this.errorItemValue = true;
            this._cd.markForCheck()
            return;
        }
        this.errorItemValue = false;

        this.itemFormLoading = true;
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._v2ProjectService.postVariableSetItem(this.project.key, this.variableSet.name, this.newItem));
            this._messageService.success(`Item ${this.newItem.name} created`);
        } catch (e) {
            this._messageService.error(`Unable to create item: ${ErrorUtils.print(e)}`);
            return;
        } finally {
            this.itemFormLoading = false;
            this._cd.markForCheck();
        }

        this.loadVariableSet();
    }

    async updateVariableSetItem(i: VariableSetItem) {
        this.loading = true;
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._v2ProjectService.updateVariableSetItem(this.project.key, this.variableSet.name, i.name, i));
            this._messageService.success(`Item ${i.name} updated`);
        } catch (e) {
            this._messageService.error(`Unable to update item: ${ErrorUtils.print(e)}`);
            return
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }

        this.loadVariableSet();
    }

    async deleteVariableSetItem(i: VariableSetItem) {
        this.loading = true;
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._v2ProjectService.deleteVariableSetItem(this.project.key, this.variableSet.name, i.name));
            this._messageService.success(`Item ${i.name} deleted`);
        } catch (e) {
            this._messageService.error(`Unable to delete item: ${ErrorUtils.print(e)}`);
            return;
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }

        this.loadVariableSet();
    }
}
