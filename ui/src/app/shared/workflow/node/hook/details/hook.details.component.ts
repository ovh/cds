import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { TaskExecution } from 'app/model/workflow.hook.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-hook-details',
    templateUrl: './hook.details.component.html',
    styleUrls: ['./hook.details.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeHookDetailsComponent implements OnInit, OnDestroy {
  @ViewChild('code') codemirror: any;
  @ViewChild('nodeHookDetailsModal') nodeHookDetailsModal: ModalTemplate<boolean, boolean, void>;

  modal: SuiActiveModal<boolean, boolean, void>;
  modalConfig: TemplateModalConfig<boolean, boolean, void>;
  task: TaskExecution;
  codeMirrorConfig: any;
  themeSubscription: Subscription;
  body: string;

  constructor(
    private _modalService: SuiModalService,
    private _theme: ThemeStore,
    private _cd: ChangeDetectorRef
  ) {
    this.codeMirrorConfig = {
      matchBrackets: true,
      autoCloseBrackets: true,
      mode: 'application/json',
      lineWrapping: true,
      autoRefresh: true,
      readOnly: true
    };
  }

  ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

  ngOnInit(): void {
    this.themeSubscription = this._theme.get()
        .pipe(finalize(() => this._cd.markForCheck()))
        .subscribe(t => {
      this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
      if (this.codemirror && this.codemirror.instance) {
        this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
      }
    });
  }

  show(taskExec: TaskExecution): void {
      this.task = cloneDeep(taskExec);
      let jsonBody;
      if (this.task.webhook) {
            jsonBody = atob(this.task.webhook.request_body);
      } else if (this.task.gerrit) {
          jsonBody = atob(this.task.gerrit.message);
      }
      try {
          this.body = JSON.stringify(JSON.parse(jsonBody), null, 4);
      } catch (e) {
          this.body = jsonBody;
      }

    this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeHookDetailsModal);
      this.modalConfig.size = 'large';
    this.modal = this._modalService.open(this.modalConfig);
  }
}
