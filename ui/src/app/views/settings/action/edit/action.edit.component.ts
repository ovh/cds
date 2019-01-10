import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Action, PipelineUsingAction } from '../../../../model/action.model';
import { ActionService } from '../../../../service/action/action.service';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
import { ActionEvent } from '../../../../shared/action/action.event.model';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-action-edit',
    templateUrl: './action.edit.html',
    styleUrls: ['./action.edit.scss']
})
@AutoUnsubscribe()
export class ActionEditComponent implements OnInit {
    action: Action;
    actionDoc: string;
    isAdmin: boolean;
    pipelinesUsingAction: Array<PipelineUsingAction>;
    path: Array<PathItem>;
    paramsSub: Subscription;

    constructor(
        private _actionService: ActionService,
        private _toast: ToastService, private _translate: TranslateService,
        private _route: ActivatedRoute, private _router: Router,
        private _authentificationStore: AuthentificationStore
    ) {
        if (this._authentificationStore.isConnected()) {
            this.isAdmin = this._authentificationStore.isAdmin();
        }
    }

    ngOnInit() {
        this.paramsSub = this._route.params.subscribe(params => {
            this._actionService.getAction(params['name']).subscribe(u => {
                this.action = u;
                this.actionDoc = u.name.toLowerCase().replace(' ', '-');
                if (this.isAdmin) {
                    this._actionService.getPipelinesUsingAction(params['name']).pipe(first()).subscribe(p => {
                        this.pipelinesUsingAction = p;
                    });
                }

                this.updatePath();
            });
        });
    }

    actionEvent(event: ActionEvent): void {
        event.action.loading = true;

        if (event.action.actions) {
            event.action.actions.forEach(a => {
                if (a.parameters) {
                    a.parameters.forEach(p => {
                        if (p.type === 'boolean' && !p.value) {
                            p.value = 'false';
                        }
                        p.value = p.value.toString();
                    });
                }
            });
        }
        if (event.action.parameters) {
            event.action.parameters.forEach(p => {
                if (p.type === 'boolean' && !p.value) {
                    p.value = 'false';
                }
                p.value = p.value.toString();
            });
        }

        switch (event.type) {
            case 'update':
                this._actionService.updateAction(this.action.name, event.action).subscribe(action => {
                    this._toast.success('', this._translate.instant('action_saved'));
                    this.action = action;
                }, () => {
                    this.action.loading = false;
                });
                break;
            case 'delete':
                this._actionService.deleteAction(event.action.name).subscribe(() => {
                    this._toast.success('', this._translate.instant('action_deleted'));
                    this._router.navigate(['settings', 'action']);
                }, () => {
                    this.action.loading = false;
                });
                break;
        }
    }

    updatePath() {
        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'action_list_title',
            routerLink: ['/', 'settings', 'action']
        }];

        if (this.action && this.action.id) {
            this.path.push(<PathItem>{
                text: this.action.name + ' - ' + this.action.type,
                routerLink: ['/', 'settings', 'action', this.action.name]
            });
        }
    }
}
