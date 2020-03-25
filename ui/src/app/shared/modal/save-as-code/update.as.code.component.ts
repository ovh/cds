import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Operation } from 'app/model/operation.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { ApplicationWorkflowService } from 'app/service/services.module';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { Observable, Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-update-ascode',
    templateUrl: './update-ascode.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class UpdateAsCodeComponent {
    @ViewChild('updateAsCodeModal', { static: false })
    public myModalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    @Input() project: Project;
    @Input() appName: string;
    @Input() name: string;

    dataToSave: any;
    dataType: string;
    branches: Array<string>;
    selectedBranch: string;
    commitMessage: string;
    loading: boolean;
    webworkerSub: Subscription;
    ope: Operation;
    pollingOperationSub: Subscription;

    constructor(
        private _modalService: SuiModalService,
        private _awService: ApplicationWorkflowService,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _workflowService: WorkflowService,
        private _pipService: PipelineService
    ) { }

    show(data: any, type: string) {
        this.loading = false;
        this.dataToSave = data;
        this.dataType = type;
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.myModalTemplate);
        this.modal = this._modalService.open(this.modalConfig);
        this._awService.getVCSInfos(this.project.key, this.appName, '').pipe(first())
            .subscribe(vcsinfos => {
                if (vcsinfos && vcsinfos.branches) {
                    this.branches = vcsinfos.branches.map(b => b.display_id);
                }
                this._cd.markForCheck();
            });
    }

    close() {
        this.modal.approve(true);
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

    save(): void {
        switch (this.dataType) {
            case 'workflow':
                this.loading = true;
                this._workflowService.updateAsCode(this.project.key, this.name, this.selectedBranch,
                    this.commitMessage, this.dataToSave).subscribe(o => {
                        this.ope = o;
                        this.startPollingOperation();
                    });
                break;
            case 'pipeline':
                this.loading = true;
                this._pipService.updateAsCode(this.project.key, <Pipeline>this.dataToSave,
                    this.selectedBranch, this.commitMessage).subscribe(o => {
                        this.ope = o;
                        this.startPollingOperation();
                    });
                break;
            default:
                this._toast.error('', this._translate.instant('ascode_error_unknown_type'))
        }
    }

    startPollingOperation() {
        this.pollingOperationSub = Observable.interval(1000).subscribe(() => {
            this._workflowService.getAsCodeOperation(this.project.key, this.name, this.ope.uuid)
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(o => {
                    this.ope = o;
                    if (this.ope.status > 1) {
                        this.loading = false;
                        this.pollingOperationSub.unsubscribe();
                    }
                });
        });
    }
}
