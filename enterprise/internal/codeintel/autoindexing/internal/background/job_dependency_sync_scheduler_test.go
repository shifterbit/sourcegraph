package background

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/log/logtest"
	"github.com/stretchr/testify/require"

	autoindexingshared "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/autoindexing/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/shared/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/uploads/shared"
	"github.com/sourcegraph/sourcegraph/internal/codeintel/dependencies"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/precise"
)

func init() {
	autoIndexingEnabled = func() bool { return true }
}

func TestDependencySyncSchedulerJVM(t *testing.T) {
	mockWorkerStore := NewMockWorkerStore()
	mockUploadsSvc := NewMockUploadService()
	mockDepedenciesSvc := NewMockDependenciesService()
	mockAutoindexingSvc := NewMockAutoIndexingService()
	mockExtsvcStore := NewMockExternalServiceStore()
	mockScanner := NewMockPackageReferenceScanner()
	mockUploadsSvc.ReferencesForUploadFunc.SetDefaultReturn(mockScanner, nil)
	mockUploadsSvc.GetUploadByIDFunc.SetDefaultReturn(types.Upload{ID: 42, RepositoryID: 50, Indexer: "scip-java"}, true, nil)
	mockScanner.NextFunc.PushReturn(shared.PackageReference{Package: shared.Package{DumpID: 42, Scheme: dependencies.JVMPackagesScheme, Name: "name1", Version: "v2.2.0"}}, true, nil)

	handler := dependencySyncSchedulerHandler{
		uploadsSvc:      mockUploadsSvc,
		depsSvc:         mockDepedenciesSvc,
		autoindexingSvc: mockAutoindexingSvc,
		workerStore:     mockWorkerStore,
		extsvcStore:     mockExtsvcStore,
	}

	logger := logtest.Scoped(t)
	job := autoindexingshared.DependencySyncingJob{
		UploadID: 42,
	}
	if err := handler.Handle(context.Background(), logger, job); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(mockAutoindexingSvc.InsertDependencyIndexingJobFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to InsertDependencyIndexingJob. want=%d have=%d", 1, len(mockAutoindexingSvc.InsertDependencyIndexingJobFunc.History()))
	} else {
		var kinds []string
		for _, call := range mockAutoindexingSvc.InsertDependencyIndexingJobFunc.History() {
			kinds = append(kinds, call.Arg2)
		}

		expectedKinds := []string{extsvc.KindJVMPackages}
		if diff := cmp.Diff(expectedKinds, kinds); diff != "" {
			t.Errorf("unexpected kinds (-want +got):\n%s", diff)
		}
	}

	if len(mockExtsvcStore.ListFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to extsvc.List. want=%d have=%d", 1, len(mockExtsvcStore.ListFunc.History()))
	}

	if len(mockDepedenciesSvc.UpsertDependencyReposFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to InsertCloneableDependencyRepo. want=%d have=%d", 1, len(mockDepedenciesSvc.UpsertDependencyReposFunc.History()))
	}
}

func TestDependencySyncSchedulerGomod(t *testing.T) {
	t.Skip()
	mockWorkerStore := NewMockWorkerStore()
	mockUploadsSvc := NewMockUploadService()
	mockDepedenciesSvc := NewMockDependenciesService()
	mockAutoindexingSvc := NewMockAutoIndexingService()
	mockExtsvcStore := NewMockExternalServiceStore()
	mockScanner := NewMockPackageReferenceScanner()
	mockUploadsSvc.ReferencesForUploadFunc.SetDefaultReturn(mockScanner, nil)
	mockUploadsSvc.GetUploadByIDFunc.SetDefaultReturn(types.Upload{ID: 42, RepositoryID: 50, Indexer: "lsif-go"}, true, nil)
	mockScanner.NextFunc.PushReturn(shared.PackageReference{Package: shared.Package{DumpID: 42, Scheme: "gomod", Name: "name1", Version: "v2.2.0"}}, true, nil)

	handler := dependencySyncSchedulerHandler{
		uploadsSvc:      mockUploadsSvc,
		depsSvc:         mockDepedenciesSvc,
		autoindexingSvc: mockAutoindexingSvc,
		workerStore:     mockWorkerStore,
		extsvcStore:     mockExtsvcStore,
	}

	logger := logtest.Scoped(t)
	job := autoindexingshared.DependencySyncingJob{
		UploadID: 42,
	}
	if err := handler.Handle(context.Background(), logger, job); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(mockAutoindexingSvc.InsertDependencyIndexingJobFunc.History()) != 1 {
		t.Errorf("unexpected number of calls to InsertDependencyIndexingJob. want=%d have=%d", 1, len(mockAutoindexingSvc.InsertDependencyIndexingJobFunc.History()))
	} else {
		var kinds []string
		for _, call := range mockAutoindexingSvc.InsertDependencyIndexingJobFunc.History() {
			kinds = append(kinds, call.Arg2)
		}

		expectedKinds := []string{""}

		if diff := cmp.Diff(expectedKinds, kinds); diff != "" {
			t.Errorf("unexpected kinds (-want +got):\n%s", diff)
		}
	}

	if len(mockExtsvcStore.ListFunc.History()) != 0 {
		t.Errorf("unexpected number of calls to extsvc.List. want=%d have=%d", 0, len(mockExtsvcStore.ListFunc.History()))
	}

	if len(mockDepedenciesSvc.UpsertDependencyReposFunc.History()) != 0 {
		t.Errorf("unexpected number of calls to InsertCloneableDependencyRepo. want=%d have=%d", 0, len(mockDepedenciesSvc.UpsertDependencyReposFunc.History()))
	}
}

func TestNewPackage(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   shared.Package
		out  *precise.Package
	}{
		{
			name: "jvm name normalization",
			in: shared.Package{
				Scheme:  dependencies.JVMPackagesScheme,
				Name:    "maven/junit/junit",
				Version: "4.2",
			},
			out: &precise.Package{
				Scheme:  dependencies.JVMPackagesScheme,
				Name:    "junit:junit",
				Version: "4.2",
			},
		},
		{
			name: "jvm name normalization no-op",
			in: shared.Package{
				Scheme:  dependencies.JVMPackagesScheme,
				Name:    "junit:junit",
				Version: "4.2",
			},
			out: &precise.Package{
				Scheme:  dependencies.JVMPackagesScheme,
				Name:    "junit:junit",
				Version: "4.2",
			},
		},
		{
			name: "npm no-op",
			in: shared.Package{
				Scheme:  dependencies.NpmPackagesScheme,
				Name:    "@graphql-mesh/graphql",
				Version: "0.24.0",
			},
			out: &precise.Package{
				Scheme:  dependencies.NpmPackagesScheme,
				Name:    "@graphql-mesh/graphql",
				Version: "0.24.0",
			},
		},
		{
			name: "npm bad-name",
			in: shared.Package{
				Scheme:  dependencies.NpmPackagesScheme,
				Name:    "@automapper/classes/transformer-plugin",
				Version: "0.24.0",
			},
			out: nil,
		},
		{
			name: "go no-op",
			in: shared.Package{
				Scheme:  dependencies.GoPackagesScheme,
				Name:    "github.com/tsenart/vegeta/v12",
				Version: "12.7.0",
			},
			out: &precise.Package{
				Scheme:  dependencies.GoPackagesScheme,
				Name:    "github.com/tsenart/vegeta/v12",
				Version: "12.7.0",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			have, err := newPackage(tc.in)
			want := tc.out

			if want == nil {
				require.Nil(t, have)
				require.NotNil(t, err)
				return
			}

			if diff := cmp.Diff(want, have); diff != "" {
				t.Fatalf("mismatch (-want, +have): %s", diff)
			}
		})
	}
}
