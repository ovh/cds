import { AfterViewInit, Component, ViewChild, ViewEncapsulation } from '@angular/core';
import { Messenger, VsCodeApi } from 'vscode-messenger-webview';
import { load, LoadOptions } from 'js-yaml';
import { HOST_EXTENSION } from 'vscode-messenger-common';
import { GenerateWorkflow, Parameter, WorkflowData, WorkflowRefresh, WorkflowTemplate, WorkflowTemplateGenerated } from '../../../../src/type';
import { WorkflowV2StagesGraphComponent } from 'workflow-graph';

export declare function acquireVsCodeApi(): VsCodeApi;
const vsCodeApi = acquireVsCodeApi();

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class AppComponent {
  title = 'cds.workflow.preview';

  workflow: string = '';
  workflowError: string = '';

  workflowTemplate: any = {};
  templateError: string = '';
  templatesInputs: {[key: string]: string} = {};
  collapsedParameters: boolean = true;
  
  viewMessenger: Messenger;

  @ViewChild('stageGraph') stageGraph: WorkflowV2StagesGraphComponent | undefined;

  constructor() {
    this.viewMessenger = new Messenger(vsCodeApi);
    this.viewMessenger.onNotification(WorkflowRefresh, e => {
      this.workflow =  (e as WorkflowData).workflow;
      delete this.workflowTemplate;
      this.templatesInputs = {};
      this.templateError = '';
      this.resizeGraph();
    });
    this.viewMessenger.onNotification(WorkflowTemplate, e => {
      let data = (e as WorkflowTemplate).workflowTemplate;
      if (data && data !== '') {
        this.receivedWorkflowTemplate(data);
      }
    });
    this.viewMessenger.onNotification(WorkflowTemplateGenerated, e => {
      this.workflow =  (e as WorkflowData).workflow;
      this.resizeGraph();
    });
    this.viewMessenger.start();
  }

  resizeGraph(): void {
    setTimeout(() => {
      if (this.stageGraph) {
        this.stageGraph.resize();
      }
    }, 1);
  }

  toggleTemplateParameters(): void {
    this.collapsedParameters = !this.collapsedParameters;
    this.resizeGraph();
  }

  receivedWorkflowTemplate(data: any): void {
    try {
      this.workflow = '';
      this.templateError = '';
      this.workflowTemplate = load(data, <LoadOptions>{
          onWarning: () => {}
      });
      let oldParams = structuredClone(this.templatesInputs);
      if (this.workflowTemplate['parameters']) {
        // Add new params
        this.workflowTemplate['parameters'].forEach((p: Parameter) => {
          if (!this.templatesInputs[p.key]) {
            this.templatesInputs[p.key] = oldParams[p.key]?oldParams[p.key]: '';
          }
        });
      }
    } catch (e: any) {
      this.templateError = e.message;
    }
  }

  generateWorkflow(): void {
    this.getWorkflowFromExtension();
  }

  async getWorkflowFromExtension() {
    let params: {[key: string]: string} = {};
    Object.keys(this.templatesInputs).forEach(k => {
        if (this.templatesInputs[k]) {
          if (this.templatesInputs[k].indexOf(' ') !== -1 && this.templatesInputs[k].indexOf('"') !== 0) {
            this.templatesInputs[k] = '"'+ this.templatesInputs[k] +'"';
          }
          params[k] = this.templatesInputs[k];
        }
    });
    const generatedWorkflow = await this.viewMessenger.sendRequest(GenerateWorkflow, HOST_EXTENSION, {parameters: params});
    this.workflow = generatedWorkflow.workflow;
    this.workflowError = generatedWorkflow.error;
  }

  identify(index: number, item: {key: string, value: string}) {
    return item.key;
  }
}
