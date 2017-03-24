import {Component, Input, EventEmitter, OnInit, ViewChild, Output} from '@angular/core';
import {SharedService} from '../../shared.service';
import {Project} from '../../../model/project.model';
import {CodemirrorComponent} from 'ng2-codemirror';
import {RepositoriesManager, Repository} from '../../../model/repositories.model';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';

declare var CodeMirror: any;
declare var _: any;

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
            this.repositoriesManager = _.cloneDeep(data.repositories_manager);
        }
        this.selectedRepoManager = this.repositoriesManager[0];
    }

    @Output() valueChange = new EventEmitter<string|number|boolean>();
    @Output() valueUpdating = new EventEmitter<boolean>();

    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;

    codeMirrorConfig: any;

    repositoriesManager: Array<RepositoriesManager>;
    repositories: Array<Repository>;
    selectedRepoManager: RepositoriesManager;
    selectedRepo: Repository;
    loadingRepos: boolean;


    constructor(public _sharedService: SharedService, private _repoManagerService: RepoManagerService) {
        this.codeMirrorConfig = {
            mode: 'perl',
            lineWrapping: true,
            lineNumbers: true
        };
    }

    ngOnInit(): void {
        if (this.type === 'boolean') {
            this.value = (this.value === 'true');
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
    }

    updateRepoManager(name: string): void {
        this.selectedRepoManager = this.repositoriesManager.find(r => r.name === name);
    }

    valueRepoChanged(name): void {
        this.value = this.selectedRepoManager.name + '##' + this.selectedRepo.name;
    }

    /**
     * Update list of repo when changing repo manager
     */
    updateListRepo(): void {
        if (this.selectedRepoManager) {
            this.loadingRepos = true;
            this._repoManagerService.getRepositories(this.project.key, this.selectedRepoManager.name)
                .subscribe( repos => {
                    this.repositories = repos;
                    this.loadingRepos = false;
                }, () => {
                    this.loadingRepos = false;
                });
        }
    }
}
