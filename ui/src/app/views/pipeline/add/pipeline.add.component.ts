import {Component} from '@angular/core';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {Pipeline} from '../../../model/pipeline.model';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../shared/toast/ToastService';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {Usage} from '../../../model/usage.model';

@Component({
    selector: 'app-pipeline-add',
    templateUrl: './pipeline.add.html',
    styleUrls: ['./pipeline.add.scss']
})
export class PipelineAddComponent {

    ready = false;
    loadingCreate = false;
    pipelineType: Array<string>;
    newPipeline = new Pipeline();
    selectedApplications: Array<string>;

    pipelineNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    pipPatternError = false;

    project: Project;

    constructor(private _pipStore: PipelineStore, private _translate: TranslateService, private _toast: ToastService,
                private _routeActivated: ActivatedRoute, private _router: Router) {
        this.project = this._routeActivated.snapshot.data['project'];
        this._pipStore.getPipelineType().subscribe(list => {
            this.ready = true;
            this.pipelineType = list.toArray();
            this.newPipeline.type = this.pipelineType[0];
        });
    }

    createPipeline(): void {
        if (!this.newPipeline.name) {
            return;
        }

        if (!this.pipelineNamePattern.test(this.newPipeline.name)) {
            this.pipPatternError = true;
            return;
        }

        if (this.selectedApplications && this.selectedApplications.length > 0) {
            if (!this.newPipeline.usage) {
                this.newPipeline.usage = new Usage();
            }
            this.selectedApplications.forEach(name => {
                this.newPipeline.usage.applications.push(this.project.applications.find(a => {
                    return a.name === name;
                }));
            });
        }

        this.loadingCreate = true;
        this._pipStore.createPipeline(this.project.key, this.newPipeline).subscribe(() => {
            this.loadingCreate = false;
            this._toast.success('', this._translate.instant('pipeline_added'));
            this._router.navigate(['/project', this.project.key, 'pipeline', this.newPipeline.name]);
        }, () => {
            this.loadingCreate = false;
        });

    }
}
