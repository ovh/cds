import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Parameter } from 'app/model/parameter.model';
import { NzModalRef } from 'ng-zorro-antd/modal';

@Component({
    selector: 'app-workflow-run-job-variable',
    templateUrl: './job.variable.html',
    styleUrls: ['./job.variable.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowRunJobVariableComponent implements OnInit {

    @Input() variables: Array<Parameter>;


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
        private _modal: NzModalRef,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        this.varGit = new Array<Parameter>();
        this.varCDS = new Array<Parameter>();
        this.varBuild = new Array<Parameter>();
        this.varEnvironment = new Array<Parameter>();
        this.varApplication = new Array<Parameter>();
        this.varPipeline = new Array<Parameter>();
        this.varProject = new Array<Parameter>();
        this.varParent = new Array<Parameter>();
        this.varWorkflow = new Array<Parameter>();
        if (this.variables) {
            this.variables.forEach(p => {
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
        this._cd.markForCheck();
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
        this._modal.destroy();
    }
}
