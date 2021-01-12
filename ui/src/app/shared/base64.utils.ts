
export class Base64 {
    static b64EncodeUnicode(str: string) {
        return btoa(encodeURIComponent(str).replace(/%([0-9A-F]{2})/g, (match, p1) => String.fromCharCode(parseInt(p1, 16))));
    }

    static b64DecodeUnicode(str: string) {
        return decodeURIComponent(Array.prototype.map.call(atob(str), (c: any) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2)).join(''));
    }
}
