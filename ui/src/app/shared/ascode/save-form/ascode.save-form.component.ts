import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { Operation } from 'app/model/operation.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { ApplicationWorkflowService } from 'app/service/application/application.workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { finalize, first } from 'rxjs/operators';

export class ParamData {
    branch_name: string;
    commit_message: string;
}

@Component({
    selector: 'app-ascode-save-form',
    templateUrl: './ascode.save-form.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class AsCodeSaveFormComponent implements OnInit {
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() operation: Operation;
    @Output() paramChange = new EventEmitter<ParamData>();

    loading: boolean;
    selectedBranch: string;
    commitMessage: string;
    branches: Array<string>;

    constructor(
        private _cd: ChangeDetectorRef,
        private _awService: ApplicationWorkflowService
    ) { }

    ngOnInit() {
        this.paramChange.emit(new ParamData());

        if (!this.workflow) {
            return;
        }

        let rootAppId = this.workflow.workflow_data.node.context.application_id;
        let rootApp = this.workflow.applications[rootAppId];

        this.loading = true;
        this._cd.markForCheck();
        this._awService.getVCSInfos(this.project.key, rootApp.name, '')
            .pipe(first())
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(vcsinfos => {
                if (vcsinfos && vcsinfos.branches) {
                    this.branches = vcsinfos.branches.map(b => b.display_id);
                }
            });
    }

    optionsFilter = (opts: Array<string>, query: string): Array<string> => {
        this.selectedBranch = query;
        let result = Array<string>();
        opts.forEach(o => {
            if (o.indexOf(query) > -1) {
                result.push(o);
            }
        });
        if (result.indexOf(query) === -1) {
            result.push(query);
        }
        return result;
    };

    changeParam(): void {
        this.paramChange.emit(<ParamData>{
            branch_name: this.selectedBranch,
            commit_message: this.commitMessage
        })
    }
}
