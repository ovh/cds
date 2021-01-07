import { ChangeDetectionStrategy, Component, Input, ViewChild } from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Parameter } from 'app/model/parameter.model';
import { cloneDeep } from 'lodash-es';

@Component({
    selector: 'app-workflow-run-job-variable',
    templateUrl: './job.variable.html',
    styleUrls: ['./job.variable.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowRunJobVariableComponent {
    @ViewChild('jobVariablesModal') jobVariablesModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    open: boolean;

    @Input()
    set variables(data: Array<Parameter>) {
        this.init();
        if (data) {
            cloneDeep(data).forEach(p => {
                if (p.name.indexOf('cds.proj.', 0) === 0) {
                    this.varProject.push(p);
                } else if (p.name.indexOf('cds.app.', 0) === 0) {
                    this.varApplication.push(p);
                } else if (p.name.indexOf('cds.pip.', 0) === 0) {
                    this.varPipeline.push(p);
                } else if (p.name.indexOf('cds.env.', 0) === 0) {
                    this.varEnvironment.push(p);
                } else if (p.name.indexOf('cds.parent.', 0) === 0) {
                    this.varParent.push(p);
                } else if (p.name.indexOf('cds.build.', 0) === 0) {
                    this.varBuild.push(p);
                } else if (p.name.indexOf('git.', 0) === 0) {
                    this.varGit.push(p);
                } else if (p.name.indexOf('workflow.', 0) === 0) {
                    this.varWorkflow.push(p);
                } else {
                    this.varCDS.push(p);
                }
            });
        }

        this.varProject = this.sort(this.varProject);
        this.varApplication = this.sort(this.varApplication);
        this.varPipeline = this.sort(this.varPipeline);
        this.varEnvironment = this.sort(this.varEnvironment);
        this.varParent = this.sort(this.varParent);
        this.varBuild = this.sort(this.varBuild);
        this.varGit = this.sort(this.varGit);
        this.varWorkflow = this.sort(this.varWorkflow);
        this.varCDS = this.sort(this.varCDS);
    }

    varGit: Array<Parameter>;
    varCDS: Array<Parameter>;
    varBuild: Array<Parameter>;
    varEnvironment: Array<Parameter>;
    varApplication: Array<Parameter>;
    varPipeline: Array<Parameter>;
    varProject: Array<Parameter>;
    varParent: Array<Parameter>;
    varWorkflow: Array<Parameter>;

    constructor(
        private _modalService: SuiModalService,
    ) { }

    init(): void {
        this.varGit = new Array<Parameter>();
        this.varCDS = new Array<Parameter>();
        this.varBuild = new Array<Parameter>();
        this.varEnvironment = new Array<Parameter>();
        this.varApplication = new Array<Parameter>();
        this.varPipeline = new Array<Parameter>();
        this.varProject = new Array<Parameter>();
        this.varParent = new Array<Parameter>();
        this.varWorkflow = new Array<Parameter>();
    }

    show(): void {
        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.jobVariablesModal);
        config.mustScroll = true;

        this.modal = this._modalService.open(config);
        this.modal.onApprove(() => {
            this.open = false;
        });
        this.modal.onDeny(() => {
            this.open = false;
        });
    }

    sort(params: Array<Parameter>): Array<Parameter> {
        return params.sort((p1, p2) => {
            if (p1.name > p2.name) {
                return 1;
            }
            if (p1.name < p2.name) {
                return -1;
            }
            return 0;
        });
    }

    clickClose() {
        this.modal.approve(true);
    }
}
