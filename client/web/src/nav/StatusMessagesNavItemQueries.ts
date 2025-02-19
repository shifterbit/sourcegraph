import { gql } from '@sourcegraph/http-client'

export const STATUS_MESSAGES = gql`
    query StatusMessages {
        statusMessages {
            ... on CloningProgress {
                __typename

                message
            }

            ... on IndexingProgress {
                __typename

                notIndexed
                indexed
            }

            ... on SyncError {
                __typename

                message
            }

            ... on ExternalServiceSyncError {
                __typename

                externalService {
                    id
                    displayName
                }
            }
        }
    }
`
