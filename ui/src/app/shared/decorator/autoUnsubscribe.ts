export function AutoUnsubscribe( blackList = [] ) {

    return function( constructor ) {
        const original = constructor.prototype.ngOnDestroy;

        constructor.prototype.ngOnDestroy = function() {
            for ( let prop in this ) {
                if (prop) {
                    const property = this[ prop ];
                    if ( blackList.indexOf(prop) === -1 ) {
                        if ( property && ( typeof property.unsubscribe === 'function' ) ) {
                            property.unsubscribe();
                        }
                    }
                }
            }
            return original && typeof original === 'function' && original.apply(this, arguments);
        };
    };
}
