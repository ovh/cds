import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit, SimpleChanges } from "@angular/core";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { VariableSet, VariableSetItem } from "app/model/variablesets.model";
import { ProjectService } from "app/service/project/project.service";
import { ToastService } from "app/shared/toast/ToastService";
import { finalize } from "rxjs";

@Component({
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

    constructor(private store: Store, private _cd: ChangeDetectorRef, private _toast: ToastService, private _projectService: ProjectService) {
        this.newItem = new VariableSetItem();
    }
    
    ngOnChanges(changes: SimpleChanges): void {
        if (!this.variableSet || !this.project) {
            return
        }
        this.loadVariableSet();
    }

    loadVariableSet(): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.getVariableSet(this.project.key, this.variableSet.name)
        .pipe(finalize(() => {
            this.loading = false;
            this.newItem = new VariableSetItem();
            this._cd.markForCheck();
        }))
        .subscribe(vs => {
            this.items = vs.items;
        });
    }

    createVariableSetItem(): void {
        if (!this.varsetItemPattern.test(this.newItem.name)) {
            this.errorItemName = true;
            this._cd.markForCheck();
            return;
        } else {
            this.errorItemName = false;
        }
        if (this.newItem.value === '') {
            this.errorItemValue = true;
            this._cd.markForCheck()
            return;
        } else {
            this.errorItemValue = false;
        }
        this.itemFormLoading = true;
        this._cd.markForCheck();
        this._projectService.postVariableSetItem(this.project.key, this.variableSet.name, this.newItem)
        .pipe(finalize(() => {
            this.itemFormLoading = false;
            this._cd.markForCheck();
        }))
        .subscribe(() => {
            this._toast.success('', `Item ${this.newItem.name} created`);
            this.loadVariableSet();
        });
        
    }

    deleteVariableSetItem(i: VariableSetItem): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.deleteVariableSetItem(this.project.key, this.variableSet.name, i.name)
        .pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
        .subscribe(() => {
            this._toast.success('', `Item ${i.name} deleted`);
            this.loadVariableSet();
        })
    }
}