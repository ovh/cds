const cache: Set<symbol> = new Set();

/**
 * Throttle a function. It will ignore any calls to it in the
 * timeout time since it was last called successfully.
 *
 * @param timeout in milliseconds
 */
export function throttleFunction(timeout: number): (
    target: any,
    propertyKey: string,
    descriptor: TypedPropertyDescriptor<any>,
) => void {
    return (
        target: any,
        propertyKey: string,
        descriptor: TypedPropertyDescriptor<any>,
    ) => {
        const oldMethod = descriptor.value;
        const identifier = Symbol();

        descriptor.value = function (...args: any[]): void {
            if (!cache.has(identifier)) {
                oldMethod.call(this, args);
                cache.add(identifier);
                setTimeout(() => {
                    cache.delete(identifier);
                }, timeout);
            }
        };
    };
}