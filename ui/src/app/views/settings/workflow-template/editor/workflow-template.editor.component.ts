import { Component, EventEmitter, Input, OnChanges, Output, ViewChild } from '@angular/core';
import { WorkflowTemplateError } from '../../../../model/workflow-template.model';

@Component({
  selector: 'app-workflow-template-editor',
  templateUrl: './workflow-template.editor.html',
  styleUrls: ['./workflow-template.editor.scss']
})
export class WorkflowTemplateEditorComponent implements OnChanges {
  @ViewChild('code') code: any;

  @Input() editable: boolean;
  @Input() removable: boolean;
  @Input() value: string;
  @Input() error: WorkflowTemplateError;
  @Output() changeValue = new EventEmitter<string>();
  @Output() remove = new EventEmitter();

  codeMirrorConfig: any;

  constructor() {
    this.codeMirrorConfig = this.codeMirrorConfig = {
      matchBrackets: true,
      autoCloseBrackets: true,
      mode: 'text/x-yaml',
      lineWrapping: true,
      autoRefresh: true,
      lineNumbers: true,
    };
  }

  ngOnChanges() {
    if (this.code && this.code.instance && this.code.instance.doc) {
      for (let i = 0; i < this.code.instance.lineCount(); i++) {
        this.code.instance.doc.removeLineClass(i, 'background', 'codeRemoved');
      }
      if (this.error) {
        this.code.instance.doc.addLineClass(this.error.line - 1, 'background', 'codeRemoved');
      }
    }

    this.codeMirrorConfig.readOnly = !this.editable;
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
