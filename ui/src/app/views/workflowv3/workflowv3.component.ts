import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflowv3',
    templateUrl: './workflowv3.html',
    styleUrls: ['./workflowv3.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3Component implements OnInit, OnDestroy {
    paramsRouteSubscription: Subscription;
    projectKey: string;
    workflowName: string;

    constructor(
        private _cd: ChangeDetectorRef,
        private _activatedRoute: ActivatedRoute
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.paramsRouteSubscription = this._activatedRoute.params.subscribe(params => {
            this.projectKey = params['key'];
            this.workflowName = params['workflowName'];
            this._cd.markForCheck();
        });
    }
}
