import {Component, Input, OnInit} from '@angular/core';
import {cloneDeep} from 'lodash';
import {finalize, first} from 'rxjs/operators';
import {ProjectPlatform} from '../../../../../model/platform.model';
import {Project} from '../../../../../model/project.model';
import {WorkflowHookModel} from '../../../../../model/workflow.hook.model';
import {
    WNode,
    WNodeHook,
    Workflow
} from '../../../../../model/workflow.model';
import {HookService} from '../../../../../service/hook/hook.service';

@Component({
    selector: 'app-workflow-node-hook-form',
    templateUrl: './hook.form.html',
    styleUrls: ['./hook.form.scss']
})
export class WorkflowNodeHookFormComponent implements OnInit {

    _hook: WNodeHook = new WNodeHook();
    canDelete = false;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;
    @Input('hook')
    set hook(data: WNodeHook) {
        if (data) {
            this.canDelete = true;
            this._hook = cloneDeep<WNodeHook>(data);
            if (this.hooksModel) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this.hook.hook_model_id);
                this._hook.model = this.selectedHookModel;
            }
            this.displayConfig = Object.keys(this._hook.config).length !== 0;
        }
    }
    get hook() {
        return this._hook;
    }

    hooksModel: Array<WorkflowHookModel>;
    selectedHookModel: WorkflowHookModel;
    readonly = false;
    loadingModels = true;
    displayConfig = false;
    invalidJSON = false;
    updateMode = false;
    codeMirrorConfig: any;
    selectedPlatform: ProjectPlatform;
    availablePlatforms: Array<ProjectPlatform>;

    constructor(private _hookService: HookService) { }

    updateHook(): void {
        this.hook.model = this.selectedHookModel;
        this.hook.config = cloneDeep(this.selectedHookModel.default_config);
        this.hook.hook_model_id = this.selectedHookModel.id;
        this.displayConfig = Object.keys(this.hook.config).length !== 0;
    }

    updatePlatform(): void {
        Object.keys(this.hook.config).forEach( k => {
            if (k === 'platform') {
                this.hook.config[k].value = this.selectedPlatform.name;
            } else {
                if (this.selectedPlatform.config[k]) {
                    this.hook.config[k] = cloneDeep(this.selectedPlatform.config[k])
                }
            }
        });
    }

    changeCodeMirror(code: string) {
        if (typeof code === 'string') {
            this.invalidJSON = false;
            try {
                JSON.parse(code);
            } catch (e) {
                this.invalidJSON = true;
            }
        }
    }

    ngOnInit(): void {
        this.availablePlatforms = this.project.platforms.filter(pf =>  pf.model.hook);
        if (this.hook && this.hook.config && this.hook.config['platform']) {
            this.selectedPlatform = this.project.platforms.find(pf => pf.name === this.hook.config['platform'].value);
        }
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: this.readonly,
        };
        this.loadingModels = true;
        this._hookService.getHookModel(this.project, this.workflow, this.node).pipe(
            first(),
            finalize(() => this.loadingModels = false)
        ).subscribe(mds => {
            this.hooksModel = mds;
            if (this.hook && this.hook.hook_model_id) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this.hook.hook_model_id);
                this.hook.model = this.selectedHookModel;
            }
        });
    }
}
