import { NgModule } from '@angular/core';
import { WorkflowV2StagesGraphComponent } from './stages-graph.component';
import { WorkflowV2JobsGraphComponent } from './jobs-graph.component';
import { GraphForkJoinNodeComponent } from './node/fork-join-node.components';
import { GraphGateNodeComponent } from './node/gate-node.component';
import { GraphJobNodeComponent } from './node/job-node.component';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzAvatarModule } from 'ng-zorro-antd/avatar';
import { NzToolTipModule } from 'ng-zorro-antd/tooltip';
import { CommonModule } from '@angular/common';




@NgModule({
  declarations: [
    WorkflowV2StagesGraphComponent,
    WorkflowV2JobsGraphComponent,
    GraphForkJoinNodeComponent,
    GraphGateNodeComponent,
    GraphJobNodeComponent,
  ],
  imports: [
    CommonModule,
    NzIconModule,
    NzAvatarModule,
    NzToolTipModule
  ],
  exports: [
    WorkflowV2StagesGraphComponent,
    WorkflowV2JobsGraphComponent,
    GraphForkJoinNodeComponent,
    GraphGateNodeComponent,
    GraphJobNodeComponent,
  ]
})
export class WorkflowGraphModule { }
