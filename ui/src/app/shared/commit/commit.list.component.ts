import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { Select } from '@ngxs/store';
import { Commit } from 'app/model/repositories.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { Column, ColumnType } from '../table/data-table.component';

@Component({
    selector: 'app-commit-list',
    templateUrl: './commit.list.html',
    styleUrls: ['./commit.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class CommitListComponent implements OnInit, OnDestroy {

    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;

    @Input() commits: Array<Commit>;
    columns: Column<Commit>[];

    constructor(private _cd: ChangeDetectorRef) {
        this.columns = [
            <Column<Commit>>{
                type: ColumnType.IMG_TEXT,
                name: 'commit_author',
                class: 'middle',
                selector: (commit: Commit) => ({
                        img: commit.author.avatar,
                        valueclass: 'author',
                        value: commit.author.displayName
                    })
            },
            <Column<Commit>>{
                type: ColumnType.LINK,
                name: 'commit_id',
                class: 'middle',
                selector: (commit: Commit) => {
                    let commitID = commit.id.substring(0, 7);
                    return {
                        link: commit.url,
                        value: commitID
                    };
                }
            },
            <Column<Commit>>{
                type: ColumnType.TEXT_HTML,
                name: 'commit_message',
                class: 'middle',
                selector: (commit: Commit) => commit.message,
            },
            <Column<Commit>>{
                type: ColumnType.TIME_AGO,
                name: 'commit_date',
                class: 'middle',
                selector: (commit: Commit) => commit.authorTimestamp,
            },
        ];
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        // if commits are provided by input, do not look at the noderun
        if (this.commits && this.commits.length) {
            return;
        }
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
           if (!nr) {
               return;
           }

           if (this.commits && nr.commits && this.commits.length === nr.commits.length) {
               return;
           }
           this.commits = nr.commits;
           this._cd.markForCheck();
        });
    }
}
