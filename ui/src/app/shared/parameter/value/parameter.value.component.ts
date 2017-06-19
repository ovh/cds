import {Component, Input, EventEmitter, OnInit, ViewChild, Output} from '@angular/core';
import {SharedService} from '../../shared.service';
import {Project} from '../../../model/project.model';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {RepositoriesManager, Repository} from '../../../model/repositories.model';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';
import {cloneDeep} from 'lodash';

declare var CodeMirror: any;

@Component({
    selector: 'app-parameter-value',
    templateUrl: './parameter.value.html',
    styleUrls: ['./parameter.value.scss']
})
export class ParameterValueComponent implements OnInit {

    @Input() type: string;
    @Input() value: string|number|boolean;
    @Input() editList = true;
    @Input() edit = true;
    @Input() suggest: Array<string> = new Array<string>();
    @Input() projectKey: string;
    @Input('project')
    set project(data: Project) {
        this.repositoriesManager = new Array<RepositoriesManager>();
        this.repositoriesManager.push({
            type: 'GIT',
            url: 'Git',
            id: 0,
            name: 'Git Url'
        });
        if (data && data.repositories_manager) {
            this.repositoriesManager.push(...cloneDeep(data.repositories_manager));
        }
        this.selectedRepoManager = this.repositoriesManager[0];
        if (data) {
            this.projectKey = data.key;
        }

    }

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
            lineNumbers: true
        };
    }

    ngOnInit(): void {
        if (this.type === 'boolean') {
            this.value = (this.value === 'true' || this.value === true);
        }
        if (this.type === 'list' && !this.editList) {
            this.list = (<string>this.value).split(';');
            this.value = this.list[0];
        }
    }

    valueChanged(): void {
        this.valueChange.emit(this.value);
    }

    sendValueChanged(): void {
        this.valueUpdating.emit(true);
    }

    changeCodeMirror(): void {
        this.valueChanged();
        let firstLine = String(this.value).split('\n')[0];

        if (firstLine.indexOf('FROM') !== -1) {
            this.codeMirrorConfig.mode = 'dockerfile';
        } else if (firstLine.indexOf('#!/usr/bin/perl') !== -1) {
            this.codeMirrorConfig.mode = 'perl';
        } else if (firstLine.indexOf('#!/usr/bin/python') !== -1) {
            this.codeMirrorConfig.mode = 'python';
        } else if (String(this.value).indexOf('c:\\') !== -1) {
            this.codeMirrorConfig.mode = 'powershell';
        } else if (firstLine.indexOf('#!/bin/bash') !== -1) {
            this.codeMirrorConfig.mode = 'bash';
        } else {
            this.codeMirrorConfig.mode = 'shell';
        }
        if (this.codemirror && this.codemirror.instance && this.codemirror.instance.options.mode !== this.codeMirrorConfig.mode) {
            this.codemirror.instance.setOption('mode', this.codeMirrorConfig.mode);
        }
        this.codemirror.instance.on('keyup', (cm, event) => {
           if (!cm.state.completionActive && event.keyCode !== 13) {
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
        if (this.selectedRepoManager.url !== 'Git') {
            this.updateListRepo();
        }
    }

    valueRepoChanged(name): void {
        this.value = this.selectedRepoManager.name + '##' + name;
        this.valueChanged();
    }

    /**
     * Update list of repo when changing repo manager
     */
    updateListRepo(): void {
        if (this.selectedRepoManager) {
            this.loadingRepos = true;
            delete this.selectedRepo;
            this._repoManagerService.getRepositories(this.projectKey, this.selectedRepoManager.name, false).first()
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
