import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input,
    OnInit,
    ViewChild
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { HookEntry, NodeEntry, WorkflowEntry } from 'app/model/export.entities.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkflowCoreService } from 'app/service/workflow/workflow.core.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchAsCodeWorkflow, GetWorkflow, ImportWorkflow, PreviewWorkflow } from 'app/store/workflow.action';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Validator } from 'jsonschema';
import * as yaml from 'js-yaml';

declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-sidebar-code',
    templateUrl: './sidebar.code.html',
    styleUrls: ['./sidebar.code.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowSidebarCodeComponent implements OnInit, AfterViewInit {
    @ViewChild('codeMirror', {static: false}) codemirror: any;

    // Project that contains the workflow
    @Input() project: Project;
    @Input() workflow: Workflow;
    // Flag indicate if sidebar is open
    @Input('open')
    set open(data: boolean) {
        if (data && !this.updated) {
            this.loadingGet = true;
            this.store.dispatch(new FetchAsCodeWorkflow({
                projectKey: this.project.key,
                workflowName: this.workflow.name
            })).pipe(finalize(() => {
                this.loadingGet = false;
                this._cd.markForCheck();
            }))
                .subscribe(() => this.exportedWf = this.workflow.asCode);
        }
        this._open = data;
    }
    get open() {
        return this._open;
    }
    _open = false;


    asCodeEditorSubscription: Subscription;
    codeMirrorConfig: any;
    exportedWf: string;
    updated = false;
    loading = false;
    loadingGet = true;
    previewMode = false;
    themeSubscription: Subscription;

    constructor(
        private store: Store,
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _workflowCore: WorkflowCoreService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        this.codeMirrorConfig = {
            mode: 'text/x-yaml',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            gutters: ['CodeMirror-lint-markers'],
            lint: {
                getAnnotations: this.workflowCheck
            }
        };
    }

    workflowCheck = (cm, updateLinting, options) => {
        const errors = CodeMirror.lint.yaml(cm);
        if (errors && errors.length > 0) {
            return errors;
        }
        if (!cm) {
            return [];
        }
        let v = new Validator();
        const yamlData = yaml.load(cm);
        let result = v.validate(yamlData, {"$schema": "http://json-schema.org/draft-04/schema#",
            "$ref": "#/definitions/Workflow",
            "definitions": {
                "HookEntry": {
                    "properties": {
                        "conditions": {
                            "$ref": "#/definitions/WorkflowNodeConditions",
                            "description": "Conditions to run this hook.\nhttps://ovh.github.io/cds/docs/concepts/workflow/run-conditions."
                        },
                        "config": {
                            "patternProperties": {
                                ".*": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        },
                        "ref": {
                            "type": "string"
                        },
                        "type": {
                            "type": "string",
                            "description": "Model of the hook.\nhttps://ovh.github.io/cds/docs/concepts/workflow/hooks"
                        }
                    },
                    "additionalProperties": true,
                    "type": "object"
                },
                "NodeEntry": {
                    "properties": {
                        "application": {
                            "type": "string",
                            "description": "The application to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"
                        },
                        "conditions": {
                            "$schema": "http://json-schema.org/draft-04/schema#",
                            "$ref": "#/definitions/WorkflowNodeConditions",
                            "description": "Conditions to run this node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/run-conditions."
                        },
                        "config": {
                            "patternProperties": {
                                ".*": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        },
                        "depends_on": {
                            "items": {
                                "type": "string"
                            },
                            "type": "array",
                            "description": "Names of the parent nodes, can be pipelines, forks or joins."
                        },
                        "environment": {
                            "type": "string",
                            "description": "The environment to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"
                        },
                        "integration": {
                            "type": "string",
                            "description": "The integration to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"
                        },
                        "one_at_a_time": {
                            "type": "boolean",
                            "description": "Set to true if you want to limit the execution of this node to one at a time."
                        },
                        "parameters": {
                            "patternProperties": {
                                ".*": {
                                    "type": "string"
                                }
                            },
                            "type": "object",
                            "description": "List of parameters for the workflow."
                        },
                        "payload": {
                            "patternProperties": {
                                ".*": {
                                    "additionalProperties": true,
                                    "type": "object"
                                }
                            },
                            "type": "object"
                        },
                        "permissions": {
                            "patternProperties": {
                                ".*": {
                                    "type": "integer"
                                }
                            },
                            "type": "object",
                            "description": "The permissions for the node (ex: myGroup: 7).\nhttps://ovh.github.io/cds/docs/concepts/permissions"
                        },
                        "pipeline": {
                            "type": "string",
                            "description": "The name of a pipeline used for pipeline node."
                        },
                        "trigger": {
                            "type": "string"
                        },
                        "when": {
                            "items": {
                                "type": "string"
                            },
                            "type": "array",
                            "description": "Set manual and status condition (ex: 'success')."
                        }
                    },
                    "additionalProperties": false,
                    "type": "object"
                },
                "NotificationEntry": {
                    "properties": {
                        "settings": {
                            "$schema": "http://json-schema.org/draft-04/schema#",
                            "$ref": "#/definitions/UserNotificationSettings"
                        },
                        "type": {
                            "type": "string"
                        }
                    },
                    "additionalProperties": true,
                    "type": "object"
                },
                "UserNotificationSettings": {
                    "properties": {
                        "conditions": {
                            "$ref": "#/definitions/WorkflowNodeConditions"
                        },
                        "on_failure": {
                            "type": "string"
                        },
                        "on_start": {
                            "type": "boolean"
                        },
                        "on_success": {
                            "type": "string"
                        },
                        "recipients": {
                            "items": {
                                "type": "string"
                            },
                            "type": "array"
                        },
                        "send_to_author": {
                            "type": "boolean"
                        },
                        "send_to_groups": {
                            "type": "boolean"
                        },
                        "template": {
                            "$schema": "http://json-schema.org/draft-04/schema#",
                            "$ref": "#/definitions/UserNotificationTemplate"
                        }
                    },
                    "additionalProperties": true,
                    "type": "object"
                },
                "UserNotificationTemplate": {
                    "properties": {
                        "body": {
                            "type": "string"
                        },
                        "disable_comment": {
                            "type": "boolean"
                        },
                        "subject": {
                            "type": "string"
                        }
                    },
                    "additionalProperties": true,
                    "type": "object"
                },
                "Workflow": {
                    "properties": {
                        "application": {
                            "type": "string",
                            "description": "The application to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"
                        },
                        "conditions": {
                            "$ref": "#/definitions/WorkflowNodeConditions",
                            "description": "Conditions to run this node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/run-conditions."
                        },
                        "depends_on": {
                            "items": {
                                "type": "string"
                            },
                            "type": "array",
                            "description": "Names of the parent nodes, can be pipelines, forks or joins."
                        },
                        "description": {
                            "type": "string"
                        },
                        "environment": {
                            "type": "string",
                            "description": "The environment to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"
                        },
                        "history_length": {
                            "type": "integer"
                        },
                        "hooks": {
                            "patternProperties": {
                                ".*": {
                                    "items": {
                                        "$schema": "http://json-schema.org/draft-04/schema#",
                                        "$ref": "#/definitions/HookEntry"
                                    },
                                    "type": "array"
                                }
                            },
                            "type": "object",
                            "description": "Workflow hooks list."
                        },
                        "integration": {
                            "type": "string",
                            "description": "The integration to use in the context of the node.\nhttps://ovh.github.io/cds/docs/concepts/workflow/pipeline-context"
                        },
                        "metadata": {
                            "patternProperties": {
                                ".*": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        },
                        "name": {
                            "type": "string",
                            "description": "The name of the workflow."
                        },
                        "notifications": {
                            "patternProperties": {
                                ".*": {
                                    "items": {
                                        "$ref": "#/definitions/NotificationEntry"
                                    },
                                    "type": "array"
                                }
                            },
                            "type": "object"
                        },
                        "notify": {
                            "items": {
                                "$schema": "http://json-schema.org/draft-04/schema#",
                                "$ref": "#/definitions/NotificationEntry"
                            },
                            "type": "array"
                        },
                        "one_at_a_time": {
                            "type": "boolean",
                            "description": "Set to true if you want to limit the execution of this node to one at a time."
                        },
                        "parameters": {
                            "patternProperties": {
                                ".*": {
                                    "type": "string"
                                }
                            },
                            "type": "object",
                            "description": "List of parameters for the workflow."
                        },
                        "payload": {
                            "patternProperties": {
                                ".*": {
                                    "additionalProperties": true,
                                    "type": "object"
                                }
                            },
                            "type": "object"
                        },
                        "permissions": {
                            "patternProperties": {
                                ".*": {
                                    "type": "integer"
                                }
                            },
                            "type": "object",
                            "description": "The permissions for the workflow (ex: myGroup: 7).\nhttps://ovh.github.io/cds/docs/concepts/permissions"
                        },
                        "pipeline": {
                            "type": "string",
                            "description": "The name of a pipeline used for pipeline node."
                        },
                        "pipeline_hooks": {
                            "items": {
                                "$ref": "#/definitions/HookEntry"
                            },
                            "type": "array"
                        },
                        "purge_tags": {
                            "items": {
                                "type": "string"
                            },
                            "type": "array"
                        },
                        "template": {
                            "type": "string",
                            "description": "Optional path of the template used to generate the workflow."
                        },
                        "version": {
                            "type": "string",
                            "description": "Version for the yaml syntax, latest is v1.0."
                        },
                        "when": {
                            "items": {
                                "type": "string"
                            },
                            "type": "array",
                            "description": "Set manual and status condition (ex: 'success')."
                        },
                        "workflow": {
                            "patternProperties": {
                                ".*": {
                                    "$schema": "http://json-schema.org/draft-04/schema#",
                                    "$ref": "#/definitions/NodeEntry"
                                }
                            },
                            "type": "object",
                            "description": "Workflow nodes list."
                        }
                    },
                    "additionalProperties": false,
                    "type": "object"
                },
                "WorkflowNodeCondition": {
                    "properties": {
                        "operator": {
                            "type": "string"
                        },
                        "value": {
                            "type": "string"
                        },
                        "variable": {
                            "type": "string"
                        }
                    },
                    "additionalProperties": true,
                    "type": "object"
                },
                "WorkflowNodeConditions": {
                    "properties": {
                        "lua_script": {
                            "type": "string"
                        },
                        "plain": {
                            "items": {
                                "$schema": "http://json-schema.org/draft-04/schema#",
                                "$ref": "#/definitions/WorkflowNodeCondition"
                            },
                            "type": "array"
                        }
                    },
                    "additionalProperties": true,
                    "type": "object"
                }
            }});
        return this.toCodemirrorError(<[]>result.errors);
    };

    toCodemirrorError(errors: []) {
        let errs = [];
        if (errors) {
            errors.forEach(e => {
                errs.push({
                    from: {
                        ch: 1,
                        line: 1
                    },
                    message: e['message']
                });
            });
        }
        return errs;
    }

    ngOnInit(): void {
        this.asCodeEditorSubscription = this._workflowCore.getAsCodeEditor()
            .subscribe((state) => {
                if (state != null && state.save) {
                    this.save();
                }
            });

        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    ngAfterViewInit(): void {
        this.codemirror.instance.on('keyup', (cm, event) => {
            if (event.key === '@' || event.keyCode > 46 || event.keyCode === 32) {
                CodeMirror.showHint(cm, CodeMirror.hint.workflowAsCode, {
                    completeSingle: true,
                    closeCharacters: / /,
                    specialChars: '',
                    snippets: [
                        {
                            'text': new WorkflowEntry().toSnippet(),
                            'displayText': '@workflow'
                        },
                        {
                            'text': new NodeEntry().toSnippet(),
                            'displayText': '@node'
                        },
                        {
                            'text': new HookEntry().toSnippet(),
                            'displayText': '@hooks'
                        }
                    ],
                    suggests: {
                        pipelines: this.project.pipeline_names.map(n => n.name),
                        applications: this.project.application_names.map(n => n.name),
                        environments: this.project.environment_names.map(n => n.name)
                    }
                });
            }
        });
    }

    keyEvent(event: KeyboardEvent) {
        if (event.key === 's' && (event.ctrlKey || event.metaKey)) {
            this.save();
            event.preventDefault();
        }
    }

    cancel() {
        if (this.previewMode) {
            this.store.dispatch(new GetWorkflow({
                projectKey: this.project.key,
                workflowName: this.workflow.name
            })).subscribe(() => this._workflowCore.toggleAsCodeEditor({ open: false, save: false }));
            this.previewMode = false;
        } else {
            this._workflowCore.setWorkflowPreview(null);
            this._workflowCore.toggleAsCodeEditor({ open: false, save: false });
        }
        this.updated = false;
    }

    unselectAll() {
        let url = this._router.createUrlTree(['./'], {
            relativeTo: this._activatedRoute,
            queryParams: {}
        });
        this._router.navigateByUrl(url.toString());
    }

    preview() {
        this.unselectAll();
        this.loading = true;
        this.previewMode = true;
        this.store.dispatch(new PreviewWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            wfCode: this.exportedWf
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => this._workflowCore.toggleAsCodeEditor({ open: false, save: false }));
    }

    save() {
        this.unselectAll();
        this.loading = true;
        this.store.dispatch(new ImportWorkflow({
            projectKey: this.project.key,
            wfName: this.workflow.name,
            workflowCode: this.exportedWf
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this.previewMode = false;
                this.updated = false;
                this._workflowCore.toggleAsCodeEditor({ open: false, save: false });
                this._workflowCore.setWorkflowPreview(null);
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }
}
