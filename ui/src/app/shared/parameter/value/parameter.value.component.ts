import {Component, Input, EventEmitter, OnInit, ViewChild, Output, ChangeDetectorRef} from '@angular/core';
import {SharedService} from '../../shared.service';
import {Project} from '../../../model/project.model';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {RepositoriesManager, Repository} from '../../../model/repositories.model';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';
import {cloneDeep} from 'lodash';
import {Parameter} from '../../../model/parameter.model';
import {first} from 'rxjs/operators';

declare var CodeMirror: any;

@Component({
    selector: 'app-parameter-value',
    templateUrl: './parameter.value.html',
    styleUrls: ['./parameter.value.scss']
})
export class ParameterValueComponent implements OnInit {

    editableValue: string|number|boolean;
    @Input() type: string;
    @Input('value')
    set value (data: string|number|boolean) {
        this.castValue(data);
    };

    @Input() editList = true;
    @Input() edit = true;
    @Input() suggest: Array<string>;
    @Input() projectKey: string;
    @Input('ref')
    set ref(data: Parameter) {
        if (data && data.type === 'list') {
            this.refValue = (<string>data.value).split(';');
        }
    }
    refValue: Array<string>;

    @Input('project')
    set project(data: Project) {
        this.repositoriesManager = new Array<RepositoriesManager>();
        if (data && data.vcs_servers) {
            this.repositoriesManager.push(...cloneDeep(data.vcs_servers));
        }
        this.selectedRepoManager = this.repositoriesManager[0];
        if (data) {
            this.projectKey = data.key;
        }
        this.projectRo = data;
    }

    projectRo: Project;

    @Output() valueChange = new EventEmitter<string|number|boolean>();
    @Output() valueUpdating = new EventEmitter<boolean>();

    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    codeMirrorConfig: any;

    repositoriesManager: Array<RepositoriesManager>;
    repositories: Array<Repository>;
    selectedRepoManager: RepositoriesManager;
    selectedRepo: string;
    loadingRepos: boolean;
    connectRepos: boolean;

    list: Array<string>;

    constructor(public _sharedService: SharedService, private _repoManagerService: RepoManagerService) {
        this.codeMirrorConfig = {
            mode: 'shell',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    ngOnInit(): void {
        this.castValue(this.editableValue);
        if (!this.suggest) {
            this.suggest = new Array<string>();
        }
        setTimeout(() => {
            this.valueChange.emit(this.editableValue);
        }, 1);

    }

    castValue(data: string|number|boolean): string|number|boolean {
        if (this.type === 'boolean') {
            this.editableValue = (data === 'true' || data === true);
            return;
        } else if (this.type === 'list') {
            if (this.editList) {
                this.editableValue = data;
            } else if (!this.list) {
                this.list = (<string>data).split(';');
                this.editableValue = this.list[0];
            }
        } else {
            this.editableValue = data;
        }
    }

    valueChanged(): void {
        this.valueChange.emit(this.editableValue);
    }

    sendValueChanged(): void {
        this.valueUpdating.emit(true);
    }

    changeCodeMirror(): void {
        this.valueChanged();
        let firstLine = String(this.editableValue).split('\n')[0];

        if (firstLine.indexOf('FROM') !== -1) {
            this.codeMirrorConfig.mode = 'dockerfile';
        } else if (firstLine.indexOf('#!/usr/bin/perl') !== -1) {
            this.codeMirrorConfig.mode = 'perl';
        } else if (firstLine.indexOf('#!/usr/bin/python') !== -1) {
            this.codeMirrorConfig.mode = 'python';
        } else if (String(this.editableValue).indexOf('c:\\') !== -1) {
            this.codeMirrorConfig.mode = 'powershell';
        } else if (firstLine.indexOf('#!/bin/bash') !== -1) {
            this.codeMirrorConfig.mode = 'shell';
        } else {
            this.codeMirrorConfig.mode = 'shell';
        }
        if (this.codemirror && this.codemirror.instance && this.codemirror.instance.options.mode !== this.codeMirrorConfig.mode) {
            this.codemirror.instance.setOption('mode', this.codeMirrorConfig.mode);
        }
        this.codemirror.instance.on('keyup', (cm, event) => {
            if (!cm.state.completionActive && (event.keyCode > 46 || event.keyCode === 32)) {
                CodeMirror.showHint(cm, CodeMirror.hint.cds, {
                    completeSingle: true,
                    closeCharacters: / /,
                    cdsCompletionList: this.suggest,
                    specialChars: ''
                });
            }
        });
    }

    updateRepoManager(name: string): void {
        this.selectedRepoManager = this.repositoriesManager.find(r => r.name === name);
        this.updateListRepo();
    }

    valueRepoChanged(name): void {
        this.editableValue = this.selectedRepoManager.name + '##' + name;
        this.valueChanged();
    }

    /**
     * Update list of repo when changing repo manager
     */
    updateListRepo(): void {
        if (this.selectedRepoManager) {
            this.loadingRepos = true;
            delete this.selectedRepo;
            this._repoManagerService.getRepositories(this.projectKey, this.selectedRepoManager.name, false).pipe(first())
                .subscribe( repos => {
                    this.selectedRepo = repos[0].fullname;
                    this.repositories = repos;
                    this.loadingRepos = false;
                }, () => {
                    this.loadingRepos = false;
                });
        }
    }
}
