export type ResolveResult<T> = { found: T } | { error: string };

/**
 * Resolve an item by full ID, short ID prefix (3+ chars), or name.
 *
 * Resolution order:
 * 1. Exact full ID match (map key lookup)
 * 2. Short ID prefix match (3+ chars, must be unambiguous)
 * 3. Name match (strips leading `/` for container names)
 */
export function resolveByIdOrName<T>(
    items: Map<string, T>,
    query: string,
    getName: (item: T) => string,
    getId: (item: T) => string,
): ResolveResult<T> {
    // 1. Exact full ID match
    const exact = items.get(query);
    if (exact) {
        return { found: exact };
    }

    // 2. Short ID prefix match (3+ chars)
    if (query.length >= 3) {
        const prefixMatches: T[] = [];
        for (const item of items.values()) {
            if (getId(item).startsWith(query)) {
                prefixMatches.push(item);
            }
        }
        if (prefixMatches.length === 1) {
            return { found: prefixMatches[0] };
        }
        if (prefixMatches.length > 1) {
            return { error: `multiple items match prefix "${query}"` };
        }
    }

    // 3. Name match (strip leading `/` if present)
    const normalizedQuery = query.startsWith("/") ? query.slice(1) : query;
    for (const item of items.values()) {
        const name = getName(item);
        const normalizedName = name.startsWith("/") ? name.slice(1) : name;
        if (normalizedName === normalizedQuery) {
            return { found: item };
        }
    }

    return { error: `no item found with id or name "${query}"` };
}
