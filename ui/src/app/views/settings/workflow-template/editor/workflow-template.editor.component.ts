import { Component, EventEmitter, Input, OnChanges, OnInit, Output, ViewChild } from '@angular/core';
import { Subscription } from 'rxjs';
import { WorkflowTemplateError } from '../../../../model/workflow-template.model';
import { ThemeStore } from '../../../../service/services.module';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-template-editor',
    templateUrl: './workflow-template.editor.html',
    styleUrls: ['./workflow-template.editor.scss']
})
@AutoUnsubscribe()
export class WorkflowTemplateEditorComponent implements OnInit, OnChanges {
    @ViewChild('code', {static: true}) codemirror: any;

    @Input() editable: boolean;
    @Input() removable: boolean;
    @Input() value: string;
    @Input() error: WorkflowTemplateError;
    @Output() changeValue = new EventEmitter<string>();
    @Output() remove = new EventEmitter();

    codeMirrorConfig: any;
    themeSubscription: Subscription;

    constructor(
        private _theme: ThemeStore
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'text/x-yaml',
            lineWrapping: true,
            autoRefresh: true,
            lineNumbers: true
        };
    }

    ngOnInit() {
        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    ngOnChanges() {
        if (this.codemirror && this.codemirror.instance && this.codemirror.instance.doc) {
            for (let i = 0; i < this.codemirror.instance.lineCount(); i++) {
                this.codemirror.instance.doc.removeLineClass(i, 'background', 'codeRemoved');
            }
            if (this.error) {
                this.codemirror.instance.doc.addLineClass(this.error.line - 1, 'background', 'codeRemoved');
            }
        }

        this.codeMirrorConfig.readOnly = !this.editable;

        if (this.codemirror && this.codemirror.instance) {
            this.codemirror.instance.setOption('readOnly', this.codeMirrorConfig.readOnly);
        }
    }

    valueChange(v: any) {
        if (typeof (v) === 'string') {
            this.changeValue.emit(v);
        }
    }

    clickRemove() {
        this.remove.emit();
    }
}
