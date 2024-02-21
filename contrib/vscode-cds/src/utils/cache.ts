
type CacheItem<T = any> = {
    value: T;
    validUntil: number;
};

type CacheStore = { [key: string]: CacheItem };

export class Cache {
    static readonly TTL_INFINITE = 0;
    static readonly TTL_SECOND = 1000;
    static readonly TTL_MINUTE = Cache.TTL_SECOND * 60;
    static readonly TTL_HOUR = Cache.TTL_MINUTE * 60;

    static set(key: string, value: any, ttl: number) {
        this.getInstance().set(key, value, ttl);
    }

    static get<T = any>(key: string): T | undefined {
        return this.getInstance().get<T>(key);
    }

    static clear() {
        this.getInstance().clear();
    }

    private static instance = new Cache();

    private static getInstance() {
        return Cache.instance;
    }

    private items: CacheStore = {}

    private constructor() { }

    private get<T = any>(key: string): T | undefined {
        const item = this.items[key] as CacheItem<T> | undefined;

        if (!item) {
            return undefined;
        }

        if (item.validUntil && item.validUntil < Date.now()) {
            return undefined;
        }

        return item.value;
    }

    private set<T = any>(key: string, value: T, ttl: number) {
        this.items[key] = {
            value,
            validUntil: ttl ? Date.now() + ttl : 0,
        };
    }

    private clear() {
        this.items = {};
    }
}
