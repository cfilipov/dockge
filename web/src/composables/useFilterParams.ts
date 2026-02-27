import { watch, type Ref } from "vue";
import { useRoute, useRouter } from "vue-router";

export interface FilterParamBinding {
    param: string;
    category: { selected: Set<string> };
}

/**
 * Two-way sync between reactive filter state and URL query params.
 * On setup, hydrates searchText + filter categories from the current URL.
 * On changes, writes state back via router.replace() (no history entries).
 */
export function useFilterParams(
    searchText: Ref<string>,
    bindings: FilterParamBinding[]
) {
    const route = useRoute();
    const router = useRouter();

    // Hydrate from URL
    const q = route.query.q;
    if (typeof q === "string" && q) {
        searchText.value = q;
    }

    for (const { param, category } of bindings) {
        const val = route.query[param];
        const str = Array.isArray(val) ? val[0] : val;
        if (typeof str === "string" && str) {
            for (const v of str.split(",")) {
                category.selected.add(v);
            }
        }
    }

    // Sync state → URL
    watch(
        () => {
            const parts: string[] = [searchText.value];
            for (const { category } of bindings) {
                parts.push([...category.selected].join(","));
            }
            return parts.join("\0");
        },
        () => {
            const query: Record<string, string> = {};
            if (searchText.value) {
                query.q = searchText.value;
            }
            for (const { param, category } of bindings) {
                if (category.selected.size > 0) {
                    query[param] = [...category.selected].join(",");
                }
            }
            router.replace({ query });
        }
    );
}
