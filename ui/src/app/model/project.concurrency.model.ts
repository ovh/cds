export class Concurrency {
    id: string;
    name: string;
    description: string;
    pool: number;
    cancel_in_progress: boolean;
    order: string;
   
    constructor() {
        this.name = '';
        this.pool = 1;
        this.order = ConcurrencyOrder.OLDEST_FIRST;
    }
}

export class ConcurrencyOrder {
    static OLDEST_FIRST = 'oldest_first';
    static NEWEST_FIRST = 'newest_first';

    static array(): Array<string> {
        return [ConcurrencyOrder.OLDEST_FIRST, ConcurrencyOrder.NEWEST_FIRST]
    }
}