import { Location } from '@angular/common';
import { AfterViewInit, Component, ElementRef, Input, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import {
    Workflow,
    WorkflowNodeHookConfigValue,
    WorkflowNodeOutgoingHook
} from 'app/model/workflow.model';
import { WorkflowNodeOutgoingHookRun } from 'app/model/workflow.run.model';
import { WorkflowEventStore } from 'app/service/services.module';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-node-outgoinghook',
    templateUrl: './outgoinghook.html',
    styleUrls: ['./outgoinghook.scss']
})
export class WorkflowNodeOutgoingHookComponent implements OnInit, AfterViewInit {

    _hook: WorkflowNodeOutgoingHook;

    @Input('hook')
    set hook(data: WorkflowNodeOutgoingHook) {
        if (data) {
            this._hook = data;
            if (this._hook.config['hookIcon']) {
                this.icon = (<WorkflowNodeHookConfigValue>this._hook.config['hookIcon']).value.toLowerCase();
            } else {
                this.icon = this._hook.model.icon.toLowerCase();
            }
        }
    }
    get hook() {
      return this._hook;
    }
    @Input() readonly = false;
    @Input() workflow: Workflow;
    @Input() project: Project;

    icon: string;
    loading = false;
    isSelected = false;
    subSelect: Subscription;
    selectedHookID: number;
    currentHookRun: WorkflowNodeOutgoingHookRun;
    subCurrentHookRun: Subscription;
    pipelineStatus = PipelineStatus;
    ready = false;

    constructor(
        private elementRef: ElementRef,
        private _workflowEventStore: WorkflowEventStore,
        private _router: Router,
        private _activatedRoute: ActivatedRoute,
        private _location: Location
    ) {}

    ngOnInit() {
        if (this._activatedRoute.snapshot.queryParams['outgoing_id']) {
            this.selectedHookID = parseInt(this._activatedRoute.snapshot.queryParams['outgoing_id'], 10);
        }

        this.subSelect = this._workflowEventStore.selectedOutgoingHook().subscribe(h => {
            if (this.hook && h) {
                this.isSelected = h.id === this.hook.id;
                return;
            }
            this.isSelected = false;
        });

        this.subCurrentHookRun = this._workflowEventStore.selectedRun().subscribe(
            wr => {
                this.currentHookRun = null;
                if (!this.hook) { return }
                if (!wr) { return }
                if (!wr.outgoing_hooks) { return }
                if (!wr.outgoing_hooks[this.hook.id]) { return }
                if (wr.outgoing_hooks[this.hook.id].length === 0) { return }
                this.currentHookRun = wr.outgoing_hooks[this.hook.id][0];

                if (!this.ready && this.hook && this.selectedHookID && this.hook.id === this.selectedHookID) {
                    this._workflowEventStore.setSelectedOutgoingHook(this.hook);
                }

                this.ready = true;
            }
        );

    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = '5px';
    }

    openEditHookSidebar(): void {
        if (this.workflow.previewMode) {
          return;
        }
        this._workflowEventStore.setSelectedOutgoingHook(this.hook);
        let url = this._router.createUrlTree(['./'], { relativeTo: this._activatedRoute, queryParams: { 'outgoing_id': this.hook.id}});
        this._location.go(url.toString());
    }
}
