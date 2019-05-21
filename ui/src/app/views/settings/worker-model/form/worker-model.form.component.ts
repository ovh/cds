import { Component, EventEmitter, Input, OnChanges, OnInit, Output, ViewChild } from '@angular/core';
import { Group } from 'app/model/group.model';
import { User } from 'app/model/user.model';
import { ModelPattern, WorkerModel } from 'app/model/worker-model.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { SharedService } from 'app/shared/shared.service';
import omit from 'lodash-es/omit';
import { finalize } from 'rxjs/operators/finalize';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-worker-model-form',
    templateUrl: './worker-model.form.html',
    styleUrls: ['./worker-model.form.scss']
})
@AutoUnsubscribe()
export class WorkerModelFormComponent implements OnInit, OnChanges {
    @ViewChild('codeMirror') codemirror: any;

    @Input() workerModel: WorkerModel;
    @Input() loading: boolean;
    @Input() currentUser: User;
    @Input() types: Array<string>;
    @Input() communications: Array<string>;
    @Input() groups: Array<Group>;
    @Input() patterns: Array<ModelPattern>;
    @Output() save = new EventEmitter();
    @Output() saveAsCode = new EventEmitter();
    @Output() delete = new EventEmitter();

    codeMirrorConfig: any;
    asCode = false;
    loadingAsCode = false;
    workerModelAsCode: string;
    patternsFiltered: Array<ModelPattern>;
    patternSelected: ModelPattern;
    descriptionRows: number;
    envNames: Array<string> = [];
    newEnvName: string;
    newEnvValue: string;
    themeSubscription: Subscription;

    constructor(
        private _sharedService: SharedService,
        private _workerModelService: WorkerModelService,
        private _theme: ThemeStore
    ) {
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };
    }

    ngOnInit(): void {
        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    ngOnChanges(): void {
        if (this.workerModel && this.workerModel.model_docker && this.workerModel.model_docker.envs) {
            this.envNames = Object.keys(this.workerModel.model_docker.envs);
        }
    }

    loadAsCode(): void {
        if (this.asCode) {
            return;
        }
        this.asCode = true;

        if (!this.workerModel.id) {
            this.workerModelAsCode = `# Example of worker model as code of type Docker
name: myWorkerModel
group: mygrouptest
communication: http
image: myImage
description: ""
type: docker
pattern_name: basic_unix`;
            return;
        }

        this.loadingAsCode = true
        this._workerModelService.exportWorkerModel(this.workerModel.id)
            .pipe(finalize(() => this.loadingAsCode = false))
            .subscribe((wmStr) => this.workerModelAsCode = wmStr);
    }

    filterPatterns(type: string) {
        this.patternsFiltered = this.patterns.filter((pattern) => pattern.type === type);
    }

    descriptionChange(): void {
        this.descriptionRows = this.getDescriptionHeight();
    }

    getDescriptionHeight(): number {
        return this._sharedService.getTextAreaheight(this.workerModel.description);
    }

    typeChange(): void {
        this.patternsFiltered = this.patterns.filter((pattern) => pattern.type === this.workerModel.type);
    }

    patternChange(): void {
        if (!this.workerModel || !this.workerModel.type || !this.patternSelected) {
            return;
        }
        switch (this.workerModel.type) {
            case 'docker':
                this.workerModel.model_docker.cmd = this.patternSelected.model.cmd;
                this.workerModel.model_docker.shell = this.patternSelected.model.shell;
                this.workerModel.model_docker.envs = this.patternSelected.model.envs;
                if (this.patternSelected.model.envs) {
                    this.envNames = Object.keys(this.patternSelected.model.envs);
                }
                break
            default:
                this.workerModel.model_virtual_machine.pre_cmd = this.patternSelected.model.pre_cmd;
                this.workerModel.model_virtual_machine.cmd = this.patternSelected.model.cmd;
                this.workerModel.model_virtual_machine.post_cmd = this.patternSelected.model.post_cmd;
        }
    }

    addEnv(newEnvName: string, newEnvValue: string) {
        if (!newEnvName) {
            return;
        }
        if (!this.workerModel.model_docker.envs) {
            this.workerModel.model_docker.envs = {};
        }
        this.workerModel.model_docker.envs[newEnvName] = newEnvValue;
        this.envNames.push(newEnvName);
        this.newEnvName = '';
        this.newEnvValue = '';
    }

    deleteEnv(envName: string, index: number) {
        this.envNames.splice(index, 1);
        this.workerModel.model_docker.envs = omit(this.workerModel.model_docker.envs, envName);
    }

    formatPattern() {
        return (pattern: ModelPattern): string => pattern.name;
    }

    clickSave(): void {
        if (!this.workerModel.name) {
            return;
        }
        this.save.emit({
            ...this.workerModel,
            pattern_name: this.patternSelected ? this.patternSelected.name : null,
            group_id: Number(this.workerModel.group_id)
        });
    }

    clickSaveAsCode(): void {
        if (!this.workerModelAsCode) {
            return;
        }
        this.saveAsCode.emit(this.workerModelAsCode);
    }

    clickDelete(): void {
        this.delete.emit();
    }
}
