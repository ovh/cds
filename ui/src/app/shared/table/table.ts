
export abstract class Table {

    protected currentPage = 1;
    public nbElementsByPage = 10;

   abstract getData(): any[];

    /**
     * Get the data for the current page.
     * @returns {any[]}
     */
    getDataForCurrentPage(): any[] {
        let indexStart = 0;
        if (this.currentPage > 1) {
            indexStart = (this.currentPage - 1) * this.nbElementsByPage;
        }
        if (!this.getData()) {
            return [];
        }
        return this.getData().slice(indexStart, this.nbElementsByPage * this.currentPage);
    }

    /**
     * Calculate the number of pages
     * @returns {number}
     */
    getNbOfPages(): number {
        if (!this.getData()) {
            return 1;
        }
        return Math.ceil(this.getData().length / this.nbElementsByPage);
    }

    /**
     * Go to next page
     */
    upPage(): void {
        let maxPage = this.getNbOfPages();
        this.currentPage = (this.currentPage === maxPage) ? this.currentPage : this.currentPage + 1;
        this.getDataForCurrentPage();
    }

    /**
     * Go to previous page.
     */
    downPage(): void {
        this.currentPage = (this.currentPage === 1) ? this.currentPage : this.currentPage - 1;
        this.getDataForCurrentPage();
    }

    /**
     * Go to the given page
     * @param page Page to go
     */
    goTopage(page: number): void {
        if (page < 1 || page > this.getNbOfPages()) {
            return;
        }
        this.currentPage = page;
        this.getDataForCurrentPage();
    }
}
