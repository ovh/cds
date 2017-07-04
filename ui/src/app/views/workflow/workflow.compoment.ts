import {Component, ViewChild} from '@angular/core';
import {SemanticSidebarComponent} from 'ng-semantic/ng-semantic';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../model/project.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {Workflow} from '../../model/workflow.model';
import {WorkflowStore} from '../../service/workflow/workflow.store';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowComponent {

    project: Project;
    workflow: Workflow;
    workflowSubscription: Subscription;
    sidebarOpen: false;

    @ViewChild('invertedSidebar')
    sidebar: SemanticSidebarComponent;

    constructor(private _activatedRoute: ActivatedRoute, private _workflowStore: WorkflowStore, private _router: Router) {
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this._activatedRoute.children.forEach(c => {
           c.params.subscribe(p => {
               console.log(p);
               let workflowName = p['workflowName'];
               if (this.project.key && workflowName) {
                   if (this.workflowSubscription) {
                       this.workflowSubscription.unsubscribe();
                   }

                   if (!this.workflow) {
                       this.workflowSubscription = this._workflowStore.getWorkflows(this.project.key, workflowName).subscribe(ws => {
                           if (ws) {
                               let updatedWorkflow = ws.get(this.project.key + '-' + workflowName);
                               if (updatedWorkflow && !updatedWorkflow.externalChange) {
                                   this.workflow = updatedWorkflow;
                               }
                           }
                       }, () => {
                           this._router.navigate(['/project', this.project.key]);
                       });
                   }
               }
           }) ;
        });
    }
}
