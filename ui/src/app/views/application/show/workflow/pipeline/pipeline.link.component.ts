import {Component, Input, ViewChild} from '@angular/core';
import {Application} from '../../../../../model/application.model';
import {Project} from '../../../../../model/project.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Pipeline} from '../../../../../model/pipeline.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-pipeline-link',
    templateUrl: './pipeline.link.html',
    styleUrls: ['./pipeline.link.scss']
})
export class ApplicationPipelineLinkComponent {

    @Input() application: Application;
    @Input() project: Project;

    @ViewChild('linkPipelineModal')
    modal: SemanticModalComponent;

    selectedPipelines = new Array<string>();

    constructor(private _appStore: ApplicationStore, private _translate: TranslateService, private _toastService: ToastService) {
    }

    linkPipelines(): void {
        if (this.selectedPipelines.length === 0) {
            return this.close();
        }
        this._appStore.attachPipelines(this.project.key, this.application.name, this.selectedPipelines).subscribe(() => {
            this._toastService.success('', this._translate.instant('application_pipelines_attached'));
            this.selectedPipelines = new Array<string>();
            if (this.modal) {
                this.modal.hide();
            }
        });
    }

    getLinkablePipelines(): Array<Pipeline> {
        let pipelines = new Array<Pipeline>();
        if (this.project && this.project.pipelines && this.application) {
            if (!this.application.pipelines) {
                pipelines.push(...this.project.pipelines);
            } else {
                this.project.pipelines.forEach( p => {
                    if (!this.application.pipelines.find( appPip => {
                        return appPip.pipeline.name === p.name;
                        })) {
                        pipelines.push(p);
                    }
                });
            }
        }
        return pipelines;
    }

    show(data?: any): void {
        if (this.modal) {
            this.modal.show(data);
        }
    }

    close(): void {
        if (this.modal) {
            this.selectedPipelines = new Array<string>();
            this.modal.hide();
        }
    }
}
