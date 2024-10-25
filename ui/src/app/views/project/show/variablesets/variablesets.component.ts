import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from "@angular/core";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { VariableSet } from "app/model/variablesets.model";
import { ToastService } from "app/shared/toast/ToastService";
import { AddVariableSetInProject, DeleteVariableSetInProject, FetchVariableSetsInProject } from "app/store/project.action";
import { finalize } from "rxjs";

@Component({
    selector: 'app-project-variable-sets',
    templateUrl: './variablesets.html',
    styleUrls: ['./variablesets.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectVariableSetsComponent implements OnInit {

    @Input() project: Project;

    loading = true;
    varFormLoading = false;

    newVariableSetName: string;

    selectedVariableSet: VariableSet;

    constructor(private store: Store, private _cd: ChangeDetectorRef, private _toast: ToastService) {}

    ngOnInit(): void {
        this.store.dispatch(new FetchVariableSetsInProject({projectKey: this.project.key}))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe();
    }

    createVariableSet(): void {
        this.loading = true;
        this.store.dispatch(new AddVariableSetInProject(this.newVariableSetName))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', 'VariableSet created'));
    }

    deleteVariableSet(v: VariableSet): void {
        this.loading = true;
        this.store.dispatch(new DeleteVariableSetInProject(v))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', `VariableSet ${v.name} deleted`));
    }

    selectVariableSet(v: VariableSet): void {
        this.selectedVariableSet = v;
        this._cd.markForCheck;
    }

    unselectVariableSet(): void {
        delete this.selectedVariableSet;
    }
}