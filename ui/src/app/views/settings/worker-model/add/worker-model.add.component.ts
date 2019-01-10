import { Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { omit } from 'lodash';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Group } from '../../../../model/group.model';
import { User } from '../../../../model/user.model';
import { ModelPattern, WorkerModel } from '../../../../model/worker-model.model';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
import { GroupService } from '../../../../service/group/group.service';
import { WorkerModelService } from '../../../../service/worker-model/worker-model.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-worker-model-add',
    templateUrl: './worker-model.add.html',
    styleUrls: ['./worker-model.add.scss']
})
@AutoUnsubscribe()
export class WorkerModelAddComponent implements OnInit {
    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    codeMirrorConfig: any;

    loading = false;
    deleteLoading = false;
    workerModel: WorkerModel;
    workerModelTypes: Array<string>;
    workerModelCommunications: Array<string>;
    workerModelGroups: Array<Group>;
    workerModelPatterns: Array<ModelPattern> = [];
    workerModelPatternsFiltered: Array<ModelPattern> = [];
    patternSelected: ModelPattern;
    currentUser: User;
    canAdd = false;
    envNames: Array<string> = [];
    newEnvName: string;
    newEnvValue: string;
    private workerModelNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    workerModelPatternError = false;
    path: Array<PathItem>;
    paramsSub: Subscription;
    workerModelAsCode: string;

    constructor(
        private _workerModelService: WorkerModelService,
        private _groupService: GroupService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _authentificationStore: AuthentificationStore
    ) {
        this.currentUser = this._authentificationStore.getUser();
        this._groupService.getGroups(true).subscribe(groups => {
            this.workerModelGroups = groups;
        });
        this.loading = true;
        this._workerModelService.getWorkerModelPatterns()
            .pipe(finalize(() => this.loading = false))
            .subscribe((patterns) => {
                this.workerModelPatternsFiltered = patterns;
                this.workerModelPatterns = patterns;
            });

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'worker_model_list_title',
            routerLink: ['/', 'settings', 'worker-model']
        }, <PathItem>{
            translate: 'common_create'
        }];

        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
        };

        this.workerModelAsCode = `# Example of worker model as code of type Docker
name: myWorkerModel
group: mygrouptest
communication: http
image: myImage
description: ""
type: docker
pattern_name: basic_unix`;
    }

    ngOnInit() {
        this.paramsSub = this._route.params.subscribe(params => {
            this._workerModelService.getWorkerModelTypes().subscribe(wmt => {
                this.workerModelTypes = wmt;
            });
            this._workerModelService.getWorkerModelCommunications().subscribe(wmc => {
                this.workerModelCommunications = wmc;
            });
            this.workerModel = new WorkerModel();
        });
    }

    clickSaveUIButton(): void {
        if (!this.workerModel.name) {
            return;
        }

        if (!this.workerModelNamePattern.test(this.workerModel.name)) {
            this.workerModelPatternError = true;
            return;
        }

        // cast to int
        this.workerModel.group_id = Number(this.workerModel.group_id);
        this.workerModel.group = this.workerModelGroups.find(g => this.workerModel.group_id === g.id);

        if (this.patternSelected) {
            this.workerModel.pattern_name = this.patternSelected.name;
        }

        this.loading = true;
        this._workerModelService.createWorkerModel(this.workerModel).subscribe(wm => {
            this.loading = false;
            this._toast.success('', this._translate.instant('worker_model_saved'));
            this._router.navigate(['settings', 'worker-model', this.workerModel.name]);
        }, () => {
            this.loading = false;
        });
    }

    clickSaveAsCodeButton(): void {
        if (!this.workerModelAsCode) {
            return;
        }
        this.loading = true;
        this._workerModelService.importWorkerModel(this.workerModelAsCode, false)
            .pipe(finalize(() => this.loading = false))
            .subscribe((wm) => {
                this.workerModel = wm;
                this._router.navigate(['settings', 'worker-model', this.workerModel.name]);
            });
    }

    filterPatterns(type: string) {
        this.patternSelected = null;
        this.workerModelPatternsFiltered = this.workerModelPatterns.filter((wmp) => wmp.type === type);
    }

    preFillModel(pattern: ModelPattern) {
        if (!this.workerModel || !this.workerModel.type || !pattern) {
            return;
        }
        switch (this.workerModel.type) {
            case 'docker':
                this.workerModel.model_docker.cmd = pattern.model.cmd;
                this.workerModel.model_docker.shell = pattern.model.shell;
                this.workerModel.model_docker.envs = pattern.model.envs;
                if (pattern.model.envs) {
                    this.envNames = Object.keys(pattern.model.envs);
                }
                break
            default:
                this.workerModel.model_virtual_machine.pre_cmd = pattern.model.pre_cmd;
                this.workerModel.model_virtual_machine.cmd = pattern.model.cmd;
                this.workerModel.model_virtual_machine.post_cmd = pattern.model.post_cmd;
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
}
