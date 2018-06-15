import {Component, Input, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Application} from '../../../../../../model/application.model';
import {Project} from '../../../../../../model/project.model';
import {ApplicationStore} from '../../../../../../service/application/application.store';
import {ToastService} from '../../../../../../shared/toast/ToastService';

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

    loading = false;
    selectedPipelines = new Array<string>();

    constructor(private _appStore: ApplicationStore, private _translate: TranslateService, private _toastService: ToastService) {
    }

    linkPipelines(): void {
        if (this.selectedPipelines.length === 0) {
            return this.close();
        }
        this.loading = true;
        this._appStore.attachPipelines(this.project.key, this.application.name, this.selectedPipelines).subscribe(() => {
            this._toastService.success('', this._translate.instant('application_pipelines_attached'));
            this.selectedPipelines = new Array<string>();
            this.loading = false;
            if (this.modal) {
                this.modal.hide();
            }
        }, () => {
            this.loading = false;
        });
    }

    getLinkablePipelines(): Array<string> {
        let pipelines = new Array<string>();
        if (this.project && Array.isArray(this.project.pipeline_names) && this.application) {
            if (!this.application.pipelines) {
                pipelines.push(...this.project.pipeline_names.map((p) => p.name));
            } else {
                this.project.pipeline_names.forEach(pip => {
                    if (!this.application.pipelines.find(appPip => appPip.pipeline.name === pip.name)) {
                        pipelines.push(pip.name);
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
