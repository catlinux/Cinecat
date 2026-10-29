package io.github.catlinux.cinecat.app

data class PagedListState<T>(
    val items: List<T> = emptyList(),
    val total: Long = 0,
    val nextOffset: Int = 0,
    val loading: Boolean = false,
    val error: Boolean = false,
    val endReached: Boolean = false,
    val pageSize: Int = PAGE_SIZE
) {
    val hasMore: Boolean
        get() = !endReached && nextOffset < total

    val canLoadMore: Boolean
        get() = !loading &&
            !endReached &&
            (items.isEmpty() || nextOffset < total)

    val initialLoading: Boolean
        get() = loading && items.isEmpty()

    fun begin(): PagedListState<T>? {
        if (!canLoadMore) {
            return null
        }
        return copy(loading = true, error = false)
    }

    fun success(offset: Int, total: Long, page: List<T>): PagedListState<T> {
        val next = offset + page.size
        val merged = (items + page).distinct()
        return copy(
            items = merged,
            total = total,
            nextOffset = next,
            loading = false,
            error = false,
            endReached = page.isEmpty() || next >= total
        )
    }

    fun failure(): PagedListState<T> = copy(loading = false, error = true)

    fun reset(): PagedListState<T> = PagedListState(pageSize = pageSize)

    companion object {
        const val PAGE_SIZE = 60
    }
}
