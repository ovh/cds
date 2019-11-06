import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Commit } from 'app/model/repositories.model';
import { Column, ColumnType } from '../table/data-table.component';

@Component({
    selector: 'app-commit-list',
    templateUrl: './commit.list.html',
    styleUrls: ['./commit.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class CommitListComponent {
    @Input() commits: Array<Commit>;
    columns: Column<Commit>[];

    constructor() {
        this.columns = [
            <Column<Commit>>{
                type: ColumnType.IMG_TEXT,
                name: 'commit_author',
                class: 'middle',
                selector: (commit: Commit) => {
                    return {
                        img: commit.author.avatar,
                        valueclass: 'author',
                        value: commit.author.displayName
                    };
                }
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
}
