package janitor

import (
	"time"

	"github.com/sourcegraph/log"

	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

func NewReconcilerWorkerResetter(logger log.Logger, workerStore dbworkerstore.Store, metrics *metrics) *dbworker.Resetter {
	options := dbworker.ResetterOptions{
		Name:     "batches_reconciler_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics:  metrics.reconcilerWorkerResetterMetrics,
	}

	resetter := dbworker.NewResetter(logger, workerStore, options)
	return resetter
}

// NewBulkOperationWorkerResetter creates a dbworker.Resetter that reenqueues lost jobs
// for processing.
func NewBulkOperationWorkerResetter(logger log.Logger, workerStore dbworkerstore.Store, metrics *metrics) *dbworker.Resetter {
	options := dbworker.ResetterOptions{
		Name:     "batches_bulk_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics:  metrics.bulkProcessorWorkerResetterMetrics,
	}

	resetter := dbworker.NewResetter(logger, workerStore, options)
	return resetter
}

// NewBatchSpecWorkspaceExecutionWorkerResetter creates a dbworker.Resetter that re-enqueues
// lost batch_spec_workspace_execution_jobs for processing.
func NewBatchSpecWorkspaceExecutionWorkerResetter(logger log.Logger, workerStore dbworkerstore.Store, metrics *metrics) *dbworker.Resetter {
	options := dbworker.ResetterOptions{
		Name:     "batch_spec_workspace_execution_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics:  metrics.batchSpecWorkspaceExecutionWorkerResetterMetrics,
	}

	resetter := dbworker.NewResetter(logger, workerStore, options)
	return resetter
}

func NewBatchSpecWorkspaceResolutionWorkerResetter(logger log.Logger, workerStore dbworkerstore.Store, metrics *metrics) *dbworker.Resetter {
	options := dbworker.ResetterOptions{
		Name:     "batch_changes_batch_spec_resolution_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics:  metrics.batchSpecResolutionWorkerResetterMetrics,
	}

	resetter := dbworker.NewResetter(logger, workerStore, options)
	return resetter
}
