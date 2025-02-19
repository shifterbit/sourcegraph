import React, { useCallback } from 'react'

import * as H from 'history'
import { NavbarQueryState } from 'src/stores/navbarSearchQueryState'
import shallow from 'zustand/shallow'

import { Form } from '@sourcegraph/branded/src/components/Form'
import { TraceSpanProvider } from '@sourcegraph/observability-client'
import {
    SearchContextInputProps,
    CaseSensitivityProps,
    SearchPatternTypeProps,
    SubmitSearchParameters,
    canSubmitSearch,
    QueryState,
} from '@sourcegraph/search'
import { SearchBox } from '@sourcegraph/search-ui'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { Settings } from '@sourcegraph/shared/src/schema/settings.schema'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'

import { AuthenticatedUser } from '../../auth'
import { useFeatureFlag } from '../../featureFlags/useFeatureFlag'
import { Notices } from '../../global/Notices'
import {
    useExperimentalFeatures,
    useNavbarQueryState,
    setSearchCaseSensitivity,
    setSearchPatternType,
} from '../../stores'
import { ThemePreferenceProps } from '../../theme'
import { submitSearch } from '../helpers'
import { useRecentSearches } from '../input/useRecentSearches'

import styles from './SearchPageInput.module.scss'

interface Props
    extends SettingsCascadeProps<Settings>,
        ThemeProps,
        ThemePreferenceProps,
        TelemetryProps,
        PlatformContextProps<'settings' | 'sourcegraphURL' | 'requestGraphQL'>,
        Pick<SubmitSearchParameters, 'source'>,
        SearchContextInputProps {
    authenticatedUser: AuthenticatedUser | null
    location: H.Location
    history: H.History
    isSourcegraphDotCom: boolean
    /** Whether globbing is enabled for filters. */
    globbing: boolean
    autoFocus?: boolean
    queryState: QueryState
    setQueryState: (newState: QueryState) => void
}

const queryStateSelector = (
    state: NavbarQueryState
): Pick<CaseSensitivityProps, 'caseSensitive'> & SearchPatternTypeProps => ({
    caseSensitive: state.searchCaseSensitivity,
    patternType: state.searchPatternType,
})

export const SearchPageInput: React.FunctionComponent<React.PropsWithChildren<Props>> = (props: Props) => {
    const { caseSensitive, patternType } = useNavbarQueryState(queryStateSelector, shallow)
    const showSearchContext = useExperimentalFeatures(features => features.showSearchContext ?? false)
    const showSearchContextManagement = useExperimentalFeatures(
        features => features.showSearchContextManagement ?? false
    )
    const editorComponent = useExperimentalFeatures(features => features.editor ?? 'codemirror6')
    const applySuggestionsOnEnter =
        useExperimentalFeatures(features => features.applySearchQuerySuggestionOnEnter) ?? true

    const [showSearchHistory] = useFeatureFlag('search-input-show-history')
    const { recentSearches, addRecentSearch } = useRecentSearches()

    const submitSearchOnChange = useCallback(
        (parameters: Partial<SubmitSearchParameters> = {}) => {
            const query = props.queryState.query

            if (canSubmitSearch(query, props.selectedSearchContextSpec)) {
                submitSearch({
                    source: 'home',
                    query,
                    history: props.history,
                    patternType,
                    caseSensitive,
                    selectedSearchContextSpec: props.selectedSearchContextSpec,
                    addRecentSearch,
                    ...parameters,
                })
            }
        },
        [
            props.queryState.query,
            props.selectedSearchContextSpec,
            props.history,
            addRecentSearch,
            patternType,
            caseSensitive,
        ]
    )

    const onSubmit = useCallback(
        (event?: React.FormEvent): void => {
            event?.preventDefault()
            submitSearchOnChange()
        },
        [submitSearchOnChange]
    )

    // We want to prevent autofocus by default on devices with touch as their only input method.
    // Touch only devices result in the onscreen keyboard not showing until the input loses focus and
    // gets focused again by the user. The logic is not fool proof, but should rule out majority of cases
    // where a touch enabled device has a physical keyboard by relying on detection of a fine pointer with hover ability.
    const isTouchOnlyDevice =
        !window.matchMedia('(any-pointer:fine)').matches && window.matchMedia('(any-hover:none)').matches

    return (
        <div className="d-flex flex-row flex-shrink-past-contents">
            <Form className="flex-grow-1 flex-shrink-past-contents" onSubmit={onSubmit}>
                <div data-search-page-input-container={true} className={styles.inputContainer}>
                    <TraceSpanProvider name="SearchBox">
                        <SearchBox
                            {...props}
                            editorComponent={editorComponent}
                            showSearchContext={showSearchContext}
                            showSearchContextManagement={showSearchContextManagement}
                            caseSensitive={caseSensitive}
                            patternType={patternType}
                            setPatternType={setSearchPatternType}
                            setCaseSensitivity={setSearchCaseSensitivity}
                            queryState={props.queryState}
                            onChange={props.setQueryState}
                            onSubmit={onSubmit}
                            autoFocus={!showSearchHistory && !isTouchOnlyDevice && props.autoFocus !== false}
                            isExternalServicesUserModeAll={window.context.externalServicesUserMode === 'all'}
                            structuralSearchDisabled={
                                window.context?.experimentalFeatures?.structuralSearch === 'disabled'
                            }
                            applySuggestionsOnEnter={applySuggestionsOnEnter}
                            showCopyQueryButton={!showSearchHistory}
                            showSearchHistory={showSearchHistory}
                            recentSearches={recentSearches}
                        />
                    </TraceSpanProvider>
                </div>
                <Notices className="my-3" location="home" settingsCascade={props.settingsCascade} />
            </Form>
        </div>
    )
}
