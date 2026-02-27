import { ref, watch, nextTick, onMounted, onBeforeUnmount, type Ref, type WatchSource } from "vue";

export function useActiveScroll(
    containerRef: Ref<HTMLElement | undefined>,
    items: WatchSource,
) {
    const isActiveVisible = ref(false);
    let activeObserver: IntersectionObserver | null = null;
    let needsInitialScroll = true;

    function scrollToActive(behavior: ScrollBehavior = "smooth") {
        const container = containerRef.value;
        const el = container?.querySelector(".item.active") as HTMLElement | null;
        if (!el || !container) return;
        const cr = container.getBoundingClientRect();
        const ar = el.getBoundingClientRect();
        if (ar.top >= cr.top && ar.bottom <= cr.bottom) return;
        container.scrollTo({
            top: el.offsetTop - container.clientHeight / 2 + el.clientHeight / 2,
            behavior,
        });
    }

    function observeActive() {
        activeObserver?.disconnect();
        const container = containerRef.value;
        const active = container?.querySelector(".item.active");
        if (!active || !container) { isActiveVisible.value = false; return; }
        const cr = container.getBoundingClientRect();
        const ar = active.getBoundingClientRect();
        isActiveVisible.value = ar.bottom > cr.top && ar.top < cr.bottom;
        activeObserver = new IntersectionObserver(([entry]) => {
            isActiveVisible.value = entry.isIntersecting;
        }, { root: container, threshold: 0.1 });
        activeObserver.observe(active as Element);
    }

    watch(items, () => {
        const wasVisible = isActiveVisible.value;
        nextTick(() => {
            if (wasVisible || needsInitialScroll) {
                scrollToActive(needsInitialScroll ? "instant" : "smooth");
                if (needsInitialScroll && containerRef.value?.querySelector(".item.active")) {
                    needsInitialScroll = false;
                }
            }
            observeActive();
        });
    });

    onMounted(() => {
        needsInitialScroll = true;
        nextTick(() => {
            const container = containerRef.value;
            const active = container?.querySelector(".item.active") as HTMLElement | null;
            if (active && container) {
                container.scrollTop = active.offsetTop - container.clientHeight / 2 + active.clientHeight / 2;
                needsInitialScroll = false;
            }
            observeActive();
        });
    });

    onBeforeUnmount(() => {
        activeObserver?.disconnect();
    });

    return {
        scrollToActive,
    };
}
