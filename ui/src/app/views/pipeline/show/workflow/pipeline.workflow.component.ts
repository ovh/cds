import {Component, Input, OnInit, DoCheck} from '@angular/core';
import {Pipeline} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {Stage} from '../../../../model/stage.model';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';


declare var _: any;

@Component({
    selector: 'app-pipeline-workflow',
    templateUrl: './pipeline.workflow.html',
    styleUrls: ['./pipeline.workflow.scss']
})
export class PipelineWorkflowComponent implements DoCheck, OnInit {

    selectedStage: Stage;
    oldLastModifiedDate: number;

    @Input() project: Project;
    @Input() pipeline: Pipeline;

    constructor(private _pipelineStore: PipelineStore, private _toast: ToastService,
                private _translate: TranslateService) {
    }

    /**
     * Init selected stage + pipeline date
     */
    ngOnInit() {
        if (this.pipeline.stages && this.pipeline.stages.length > 0) {
            this.selectedStage = this.pipeline.stages[0];
        }
        this.oldLastModifiedDate = this.pipeline.last_modified;
    }

    /**
     * Update selected Stage On pipeline update.
     * Do not work with ngOnChange.
     */
    ngDoCheck() {
        if (this.pipeline.last_modified !== this.oldLastModifiedDate) {
            // If pipeline changed - update selected stage
            if (this.selectedStage && this.pipeline.stages) {
                let index = this.pipeline.stages.findIndex(s => s.id === this.selectedStage.id);
                if (index >= -1) {
                    this.selectedStage = this.pipeline.stages[index];
                } else {
                    this.selectedStage = undefined;
                }
            } else if (this.pipeline.stages && this.pipeline.stages.length > 0) {
                this.selectedStage = this.pipeline.stages[0];
            } else {
                this.selectedStage = undefined;
            }
        }
    }

    /**
     * Add a stage.
     */
    addStage(): void {
        let s = new Stage();
        s.enabled = true;
        if (!this.pipeline.stages) {
            this.pipeline.stages = new Array<Stage>();
        }

        s.name = 'Stage ' + (this.pipeline.stages.length + 1);
        this._pipelineStore.addStage(this.project.key, this.pipeline.name, s).subscribe(() => {
            this._toast.success('', this._translate.instant('step_added'));
            this.selectedStage = this.pipeline.stages[this.pipeline.stages.length - 1];
        });
    }
}
