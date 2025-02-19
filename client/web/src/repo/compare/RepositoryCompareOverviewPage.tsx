import * as React from 'react'

import { RouteComponentProps } from 'react-router'
import { merge, Observable, of, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, switchMap } from 'rxjs/operators'

import { ErrorAlert } from '@sourcegraph/branded/src/components/alerts'
import { HoverMerged } from '@sourcegraph/client-api'
import { Hoverifier } from '@sourcegraph/codeintellify'
import { asError, createAggregateError, ErrorLike, isErrorLike, logger } from '@sourcegraph/common'
import { gql } from '@sourcegraph/http-client'
import { ActionItemAction } from '@sourcegraph/shared/src/actions/ActionItem'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { FileSpec, RepoSpec, ResolvedRevisionSpec, RevisionSpec } from '@sourcegraph/shared/src/util/url'
import { LoadingSpinner, Text } from '@sourcegraph/wildcard'

import { queryGraphQL } from '../../backend/graphql'
import { PageTitle } from '../../components/PageTitle'
import { RepositoryComparisonFields, Scalars } from '../../graphql-operations'
import { eventLogger } from '../../tracking/eventLogger'

import { RepositoryCompareAreaPageProps } from './RepositoryCompareArea'
import { RepositoryCompareCommitsPage } from './RepositoryCompareCommitsPage'
import { RepositoryCompareDiffPage } from './RepositoryCompareDiffPage'

function queryRepositoryComparison(args: {
    repo: Scalars['ID']
    base: string | null
    head: string | null
}): Observable<RepositoryComparisonFields['comparison']['range']> {
    return queryGraphQL(
        gql`
            query RepositoryComparison($repo: ID!, $base: String, $head: String) {
                node(id: $repo) {
                    ...RepositoryComparisonFields
                }
            }

            fragment RepositoryComparisonFields on Repository {
                comparison(base: $base, head: $head) {
                    range {
                        expr
                        baseRevSpec {
                            object {
                                oid
                            }
                        }
                        headRevSpec {
                            object {
                                oid
                            }
                        }
                    }
                }
            }
        `,
        args
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.node) {
                throw createAggregateError(errors)
            }
            const repo = data.node as RepositoryComparisonFields
            if (
                !repo.comparison ||
                !repo.comparison.range ||
                !repo.comparison.range.baseRevSpec ||
                !repo.comparison.range.baseRevSpec.object ||
                !repo.comparison.range.headRevSpec ||
                !repo.comparison.range.headRevSpec.object ||
                errors
            ) {
                throw createAggregateError(errors)
            }
            eventLogger.log('RepositoryComparisonFetched')
            return repo.comparison.range
        })
    )
}

interface Props
    extends RepositoryCompareAreaPageProps,
        RouteComponentProps<{}>,
        PlatformContextProps,
        ExtensionsControllerProps,
        ThemeProps {
    /** The base of the comparison. */
    base: { repoName: string; repoID: Scalars['ID']; revision?: string | null }

    /** The head of the comparison. */
    head: { repoName: string; repoID: Scalars['ID']; revision?: string | null }

    /** An optional path of a specific file to compare */
    path: string | null

    hoverifier: Hoverifier<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec, HoverMerged, ActionItemAction>
}

interface State {
    /** The comparison's range, null when no comparison is requested, undefined while loading, or an error. */
    rangeOrError?: null | RepositoryComparisonFields['comparison']['range'] | ErrorLike
}

/** A page with an overview of the comparison. */
export class RepositoryCompareOverviewPage extends React.PureComponent<Props, State> {
    public state: State = {}

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('RepositoryCompareOverview')

        this.subscriptions.add(
            this.componentUpdates
                .pipe(
                    distinctUntilChanged(
                        (a, b) =>
                            a.repo.id === b.repo.id &&
                            a.base.revision === b.base.revision &&
                            a.head.revision === b.head.revision
                    ),
                    switchMap(({ repo, base, head }) => {
                        if (!base.revision && !head.revision) {
                            return of({ rangeOrError: null })
                        }
                        return merge(
                            of({ rangeOrError: undefined }),
                            queryRepositoryComparison({
                                repo: repo.id,
                                base: base.revision || null,
                                head: head.revision || null,
                            }).pipe(
                                catchError(error => [asError(error)]),
                                map((rangeOrError): Pick<State, 'rangeOrError'> => ({ rangeOrError }))
                            )
                        )
                    })
                )
                .subscribe(
                    stateUpdate => this.setState(stateUpdate),
                    error => logger.error(error)
                )
        )
        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="repository-compare-page">
                <PageTitle title="Compare" />
                {this.state.rangeOrError === null ? (
                    <Text>Your results will appear here</Text>
                ) : this.state.rangeOrError === undefined ? (
                    <LoadingSpinner />
                ) : isErrorLike(this.state.rangeOrError) ? (
                    <ErrorAlert className="mt-2" error={this.state.rangeOrError} />
                ) : (
                    <>
                        <RepositoryCompareCommitsPage {...this.props} />
                        <div className="mb-3" />
                        <RepositoryCompareDiffPage
                            {...this.props}
                            base={{
                                repoName: this.props.base.repoName,
                                repoID: this.props.base.repoID,
                                revision: this.props.base.revision || null,
                                commitID: this.state.rangeOrError.baseRevSpec.object!.oid,
                            }}
                            head={{
                                repoName: this.props.head.repoName,
                                repoID: this.props.head.repoID,
                                revision: this.props.head.revision || null,
                                commitID: this.state.rangeOrError.headRevSpec.object!.oid,
                            }}
                            extensionsController={this.props.extensionsController}
                        />
                    </>
                )}
            </div>
        )
    }
}
