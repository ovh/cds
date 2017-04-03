import {Component, Input, OnInit, DoCheck} from '@angular/core';
import {Pipeline} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {Stage} from '../../../../model/stage.model';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {DragulaService} from 'ng2-dragula/components/dragula.provider';


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
                private _translate: TranslateService, private dragulaService: DragulaService) {
        dragulaService.setOptions('bag-stage', {
            moves: function (el, source, handle) {
                return handle.classList.contains('move');
            },
            accepts: function(el, target, source, sibling) {
                if (sibling === null) {
                    return false;
                }
                return true;
            }
        });
        dragulaService.drop.subscribe( v => {
            setTimeout(() => {
                let stageMovedBuildOrder = Number(v[1].id.replace('step', ''));
                let stageMoved: Stage;
                for (let i = 0; i < this.pipeline.stages.length; i++) {
                    if (this.pipeline.stages[i].build_order === stageMovedBuildOrder) {
                        stageMoved = this.pipeline.stages[i];
                        stageMoved.build_order = i + 1;
                        break;
                    }
                }
                this._pipelineStore.moveStage(this.project.key, this.pipeline.name, stageMoved).subscribe(() => {
                    this._toast.success('', this._translate.instant('pipeline_stage_moved'));
                });
            });
        });
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
