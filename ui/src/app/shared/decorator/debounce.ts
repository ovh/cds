import debounce from 'lodash-es/debounce';

export default function Debounce(delay: number) {
    return function (target: any, key: any, descriptor: any) {
        const oldFunc = descriptor.value;
        const newFunc = debounce(oldFunc, delay);
        descriptor.value = function () {
            return newFunc.apply(this, arguments);
        }
    }
}

