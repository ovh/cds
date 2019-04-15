import {Component, Input, ViewChild} from '@angular/core';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {finalize, flatMap} from 'rxjs/operators';
import {Store} from '@ngxs/store';
import {SuiModalService} from 'ng2-semantic-ui';

import {cloneDeep} from 'lodash';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {WNode, Workflow} from 'app/model/workflow.model';
import {Project} from 'app/model/project.model';
import {Pipeline} from 'app/model/pipeline.model';
import {PermissionValue} from 'app/model/permission.model';
import {ApplicationWorkflowService, VariableService} from 'app/service/services.module';
import {FetchPipeline} from 'app/store/pipelines.action';
import {PipelinesState} from 'app/store/pipelines.state';
import {ParameterEvent} from 'app/shared/parameter/parameter.event.model';
import {Application} from 'app/model/application.model';

declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-node-input',
    templateUrl: './wizard.input.html',
    styleUrls: ['./wizard.input.scss']
})
@AutoUnsubscribe()
export class WorkflowWizardNodeInputComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    editableNode: WNode;

    @Input('node') set node(data: WNode) {
        if (data) {
            this.init(data);
        }
    };

    get node(): WNode {
        return this.editableNode;
    }

    @ViewChild('textareaCodeMirror')
    codemirror: CodemirrorComponent;

    suggest: string[] = [];
    payloadString: string;
    branches: string[] = [];
    permissionEnum = PermissionValue;
    invalidJSON = false;
    loadingBranches = false;
    codeMirrorConfig: {};
    pipParamsReady = false;
    currentPipeline: Pipeline;
    remotes: string[] = [];
    tags: string[] = [];

    constructor(
        private store: Store,
        private _variableService: VariableService,
        private _modalService: SuiModalService,
        private _appWorkflowService: ApplicationWorkflowService
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true
        };
    }

    init(data: WNode): void {
        this.editableNode = cloneDeep(data);
        if (!this.editableNode.context.default_payload) {
            this.editableNode.context.default_payload = {};
        }

        this.suggest = [];
        this._variableService.getContextVariable(this.project.key, this.node.context.pipeline_id)
            .subscribe((suggest) => this.suggest = suggest);

        // TODO delete .repository_fullname condition and update handler to get history branches of node_run (issue: #1815)
        let app = Workflow.getApplication(this.workflow, this.node);
        if (this.node.context && app && app.repository_fullname) {
            this.loadingBranches = true;
            this.refreshVCSInfos(app);
        }


        this.payloadString = JSON.stringify(this.editableNode.context.default_payload, undefined, 4);


        let pipeline = Workflow.getPipeline(this.workflow, this.node);
        if (pipeline) {
            this.store.dispatch(new FetchPipeline({
                projectKey: this.project.key,
                pipelineName: pipeline.name
            })).pipe(
                flatMap(() => this.store.selectOnce(PipelinesState.selectPipeline(this.project.key, pipeline.name)))
            ).subscribe((pip) => {
                this.currentPipeline = pip;
                this.pipParamsReady = true;
                this.editableNode.context.default_pipeline_parameters =
                    cloneDeep(Pipeline.mergeAndKeepOld(pip.parameters, this.editableNode.context.default_pipeline_parameters));
                try {
                    this.editableNode.context.default_payload = JSON.parse(this.payloadString);
                    this.invalidJSON = false;
                } catch (e) {
                    this.invalidJSON = true;
                }
                if (!this.editableNode.context.default_payload) {
                    this.editableNode.context.default_payload = {};
                }
            });
        }
    }


    reindent(): void {
        this.updateValue(this.payloadString);
    }

    parameterEvent(event: ParameterEvent) {
        switch (event.type) {
            case 'delete':
                this.editableNode.context.default_pipeline_parameters =
                    this.editableNode.context.default_pipeline_parameters.filter((param) => param.name !== event.parameter.name);
                event.parameter.updating = false;
                break;
        }
    }

    changeCodeMirror(eventRoot: Event): void {
        if (eventRoot.type !== 'click') {
            this.updateValue(eventRoot);
        }
        if (!this.codemirror || !this.codemirror.instance) {
            return;
        }

        if (eventRoot.type === 'click') {
            this.showHint(this.codemirror.instance, null);
        }
        this.codemirror.instance.on('keyup', (cm, event) => {
            if (!cm.state.completionActive && event.keyCode !== 32) {
                this.showHint(cm, event);
            }
        });
    }

    refreshVCSInfos(app: Application, remote?: string) {
        this._appWorkflowService.getVCSInfos(this.project.key, app.name, remote)
            .pipe(finalize(() => this.loadingBranches = false))
            .subscribe((vcsInfos) => {
                if (vcsInfos.branches) {
                    this.branches = vcsInfos.branches.map((br) => '"' + br.display_id + '"');
                }
                if (vcsInfos.remotes) {
                    this.remotes = vcsInfos.remotes.map((rem) => '"' + rem.fullname + '"');
                }
                if (vcsInfos.tags) {
                    this.tags = vcsInfos.tags.map((tag) => '"' + tag.tag + '"');
                }
            });
    }


    updateValue(payload): void {
        let newPayload: {};
        if (!payload) {
            return;
        }

        try {
            newPayload = JSON.parse(payload);
            this.invalidJSON = false;
        } catch (e) {
            this.invalidJSON = true;
            return;
        }
        this.payloadString = JSON.stringify(newPayload, undefined, 4);
        this.editableNode.context.default_payload = JSON.parse(this.payloadString);
    }

    showHint(cm, event) {
        CodeMirror.showHint(this.codemirror.instance, CodeMirror.hint.payload, {
            completeSingle: true,
            closeCharacters: / /,
            payloadCompletionList: {
                branches: this.branches,
                repositories: this.remotes,
                tags: this.tags,
            },
            specialChars: ''
        });
    }
}
