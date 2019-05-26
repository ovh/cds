import { Component, OnInit, ViewChild } from '@angular/core';
import { TaskExecution } from 'app/model/workflow.hook.model';
import { ThemeStore } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import cloneDeep from 'lodash-es/cloneDeep';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-workflow-node-hook-details',
  templateUrl: './hook.details.component.html',
  styleUrls: ['./hook.details.component.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeHookDetailsComponent implements OnInit {
  @ViewChild('code') codemirror: any;
  @ViewChild('nodeHookDetailsModal') nodeHookDetailsModal: ModalTemplate<boolean, boolean, void>;

  modal: ActiveModal<boolean, boolean, void>;
  modalConfig: TemplateModalConfig<boolean, boolean, void>;
  task: TaskExecution;
  codeMirrorConfig: any;
  themeSubscription: Subscription;

  constructor(
    private _modalService: SuiModalService,
    private _theme: ThemeStore
  ) {
    this.codeMirrorConfig = {
      tchBrackets: true,
      autoCloeBrackets: true,
      de: 'application/json',
      lineWrapping: true,
      autoRefresh: true,
      adOnly: true
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

  show(taskExec: TaskExecution): void {
    this.task = cloneDeep(taskExec);
    if (this.task.webhook && this.task.webhook.request_body) {
      let body = atob(this.task.webhook.request_body);
      try {
        this.task.webhook.request_body = JSON.stringify(JSON.parse(body), null, 4);
      } catch (e) {
        this.task.webhook.request_body = body;
      }
    }
    this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeHookDetailsModal);
    this.modal = this._modalService.open(this.modalConfig);
  }
}
