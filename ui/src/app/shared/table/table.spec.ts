/* eslint-disable @typescript-eslint/no-unused-vars */
import { fakeAsync, TestBed } from '@angular/core/testing';
import { Table } from './table';

describe('CDS: Table component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
            ],
            imports: [
            ]
        }).compileComponents();

    });

    it('Table method', fakeAsync(() => {
        // Create loginComponent
        let myTable = new MyTable();
        myTable.nbElementsByPage = 2;

        expect(myTable.getNbOfPages()).toBe(3);
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['aa', 'bb']));
        myTable.goTopage(2);
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['cc', 'dd']));
        myTable.upPage();
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['ee']));
        myTable.upPage();
        myTable.upPage();
        myTable.upPage();
        myTable.upPage();
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['ee']));
        myTable.downPage();
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['cc', 'dd']));
        myTable.downPage();
        myTable.downPage();
        myTable.downPage();
        myTable.downPage();
        myTable.downPage();
        myTable.downPage();
        myTable.downPage();
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['aa', 'bb']));

        expect(myTable.goTopage(-2));
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['aa', 'bb']));

        expect(myTable.goTopage(999));
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify(['aa', 'bb']));
    }));

    it('Null Table', fakeAsync(() => {
        // Create loginComponent
        let myTable = new MyEmptyTable();
        expect(JSON.stringify(myTable.getDataForCurrentPage())).toBe(JSON.stringify([]));
        expect(myTable.getNbOfPages()).toBe(1);
    }));

    class MyTable extends Table<string> {
        getData(): Array<string> {
            return [
                'aa',
                'bb',
                'cc',
                'dd',
                'ee'
            ];
        }
    }

    class MyEmptyTable extends Table<string> {
        getData(): Array<string> {
            return null;
        }
    }
});

