import { SubmitSearchParameters } from '@sourcegraph/search'
import { appendContextFilter } from '@sourcegraph/shared/src/search/query/transformer'
import { buildSearchURLQuery } from '@sourcegraph/shared/src/util/url'

import { eventLogger } from '../tracking/eventLogger'

import { AGGREGATION_MODE_URL_KEY, AGGREGATION_UI_MODE_URL_KEY } from './results/components/aggregation/constants'

/**
 * By default {@link submitSearch} overrides all existing query parameters.
 * This breaks all functionality that is built on top of URL query params and history
 * state. This list of query keys will be preserved between searches.
 */
const PRESERVED_QUERY_PARAMETERS = ['feat', 'trace', AGGREGATION_MODE_URL_KEY, AGGREGATION_UI_MODE_URL_KEY]

/**
 * Returns a URL query string with only the parameters in PRESERVED_QUERY_PARAMETERS.
 */
function preservedQuery(query: string): string {
    const old = new URLSearchParams(query)
    const filtered = new URLSearchParams()
    for (const key of PRESERVED_QUERY_PARAMETERS) {
        for (const value of old.getAll(key)) {
            filtered.append(key, value)
        }
    }
    return filtered.toString()
}

/**
 * @param activation If set, records the DidSearch activation event for the new user activation
 * flow.
 */
export function submitSearch({
    history,
    query,
    patternType,
    caseSensitive,
    selectedSearchContextSpec,
    source,
    addRecentSearch,
}: SubmitSearchParameters): void {
    let searchQueryParameter = buildSearchURLQuery(query, patternType, caseSensitive, selectedSearchContextSpec)

    const preserved = preservedQuery(history.location.search)
    if (preserved !== '') {
        searchQueryParameter = searchQueryParameter + '&' + preserved
    }

    // Go to search results page
    const path = '/search?' + searchQueryParameter

    const queryWithContext = appendContextFilter(query, selectedSearchContextSpec)
    eventLogger.log(
        'SearchSubmitted',
        {
            query: queryWithContext,
            source,
        },
        { source }
    )
    addRecentSearch?.(queryWithContext)
    history.push(path, { ...(typeof history.location.state === 'object' ? history.location.state : null), query })
}
