import { AfterViewChecked, ChangeDetectionStrategy, Component, EventEmitter, Input, OnInit, Output, ViewChild } from '@angular/core';
import { AllKeys } from 'app/model/keys.model';
import { Parameter } from 'app/model/parameter.model';
import { Project } from 'app/model/project.model';
import { RepositoriesManager, Repository } from 'app/model/repositories.model';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { SharedService } from 'app/shared/shared.service';
import cloneDeep from 'lodash-es/cloneDeep';
import { first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

declare var CodeMirror: any;

@Component({
    selector: 'app-parameter-value',
    templateUrl: './parameter.value.html',
    styleUrls: ['./parameter.value.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ParameterValueComponent implements OnInit, AfterViewChecked {

    editableValue: string | number | boolean;
    @Input() type: string;
    @Input() keys: AllKeys;
    @Input('value')
    set value(data: string | number | boolean) {
        this.castValue(data);
    };

    @Input() editList = true;
    @Input() edit = true;
    @Input() suggest: Array<string>;
    @Input() projectKey: string;

    @Input('ref')
    set ref(data: Parameter | Array<string>) {
        if (data && (<Parameter>data).type === 'list') {
            this.refValue = (<string>(<Parameter>data).value).split(';');
        } else if (data && Array.isArray(data)) {
            this.list = data;
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

    @Output() valueChange = new EventEmitter<string | number | boolean>();
    @Output() valueUpdating = new EventEmitter<boolean>();

    @ViewChild('codeMirror', { static: false }) codemirror: any;

    codeMirrorConfig: any;
    repositoriesManager: Array<RepositoriesManager>;
    repositories: Array<Repository>;
    selectedRepoManager: RepositoriesManager;
    selectedRepo: string;
    loadingRepos: boolean;
    connectRepos: boolean;
    alreadyRefreshed: boolean;
    list: Array<string>;
    themeSubscription: Subscription;

    constructor(
        private _repoManagerService: RepoManagerService,
        private _theme: ThemeStore,
        public _sharedService: SharedService // used in html
    ) {
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
        this.updateListRepo();

        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    ngAfterViewChecked(): void {
        if (this.codemirror && this.codemirror.instance && !this.alreadyRefreshed) {
            this.alreadyRefreshed = true;
            setTimeout(() => {
                this.codemirror.instance.refresh();
            }, 1);
        }
    }

    castValue(data: string | number | boolean): string | number | boolean {
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
        if (this.selectedRepoManager && this.type === 'repository') {
            this.loadingRepos = true;
            delete this.selectedRepo;
            this._repoManagerService.getRepositories(this.projectKey, this.selectedRepoManager.name, false).pipe(first())
                .subscribe(repos => {
                    this.selectedRepo = repos[0].fullname;
                    this.repositories = repos;
                    this.loadingRepos = false;
                }, () => {
                    this.loadingRepos = false;
                });
        }
    }

    keyExist(key: string): boolean {
        return this.keys.ssh.find((k) => k.name === key) != null;
    }
}
