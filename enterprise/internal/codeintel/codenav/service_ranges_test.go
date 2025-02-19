package codenav

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/go-diff/diff"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/codenav/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/shared/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	sgtypes "github.com/sourcegraph/sourcegraph/internal/types"
)

const rangesDiff = `
diff --git a/changed.go b/changed.go
index deadbeef1..deadbeef2 100644
--- a/changed.go
+++ b/changed.go
@@ -12,7 +12,7 @@ const imageProcWorkers = 1
 var imageProcSem = make(chan bool, imageProcWorkers)
 var random = "banana"

 func (i *imageResource) doWithImageConfig(conf images.ImageConfig, f func(src image.Image) (image.Image, error)) (resource.Image, error) {
-       img, err := i.getSpec().imageCache.getOrCreate(i, conf, func() (*imageResource, image.Image, error) {
+       return i.getSpec().imageCache.getOrCreate(i, conf, func() (*imageResource, image.Image, error) {
-                imageProcSem <- true
+                defer func() {
`

func TestRanges(t *testing.T) {
	// Set up mocks
	mockStore := NewMockStore()
	mockLsifStore := NewMockLsifStore()
	mockUploadSvc := NewMockUploadService()
	mockGitserverClient := NewMockGitserverClient()
	mockGitServer := NewMockGitserverClient()
	mockGitServer.DiffPathFunc.SetDefaultHook(func(ctx context.Context, srpc authz.SubRepoPermissionChecker, rn api.RepoName, sourceCommit, targetCommit, path string) ([]*diff.Hunk, error) {
		if path == "sub3/changed.go" {
			fileDiff, err := diff.ParseFileDiff([]byte(rangesDiff))
			if err != nil {
				return nil, err
			}
			return fileDiff.Hunks, nil
		}
		return nil, nil
	})

	// Init service
	svc := newService(mockStore, mockLsifStore, mockUploadSvc, mockGitserverClient, nil, nil, &observation.TestContext)

	// Set up request state
	mockRequestState := RequestState{}
	mockRequestState.SetLocalCommitCache(mockGitserverClient)
	mockRequestState.SetLocalGitTreeTranslator(mockGitServer, &sgtypes.Repo{}, mockCommit, mockPath, 50)
	uploads := []types.Dump{
		{ID: 50, Commit: "deadbeef1", Root: "sub1/", RepositoryID: 42},
		{ID: 51, Commit: "deadbeef1", Root: "sub2/", RepositoryID: 42},
		{ID: 52, Commit: "deadbeef2", Root: "sub3/", RepositoryID: 42},
		{ID: 53, Commit: "deadbeef1", Root: "sub4/", RepositoryID: 42},
	}
	mockRequestState.SetUploadsDataLoader(uploads)

	testLocation1 := shared.Location{DumpID: 50, Path: "a.go", Range: testRange1}
	testLocation2 := shared.Location{DumpID: 51, Path: "b.go", Range: testRange2}
	testLocation3 := shared.Location{DumpID: 51, Path: "c.go", Range: testRange1}
	testLocation4 := shared.Location{DumpID: 51, Path: "d.go", Range: testRange2}
	testLocation5 := shared.Location{DumpID: 51, Path: "e.go", Range: testRange1}
	testLocation6 := shared.Location{DumpID: 51, Path: "a.go", Range: testRange2}
	testLocation7 := shared.Location{DumpID: 51, Path: "a.go", Range: testRange3}
	testLocation8 := shared.Location{DumpID: 52, Path: "a.go", Range: testRange4}
	testLocation9 := shared.Location{DumpID: 52, Path: "changed.go", Range: testRange6}

	ranges := []shared.CodeIntelligenceRange{
		{Range: testRange1, HoverText: "text1", Definitions: nil, References: []shared.Location{testLocation1}, Implementations: []shared.Location{}},
		{Range: testRange2, HoverText: "text2", Definitions: []shared.Location{testLocation2}, References: []shared.Location{testLocation3}, Implementations: []shared.Location{}},
		{Range: testRange3, HoverText: "text3", Definitions: []shared.Location{testLocation4}, References: []shared.Location{testLocation5}, Implementations: []shared.Location{}},
		{Range: testRange4, HoverText: "text4", Definitions: []shared.Location{testLocation6}, References: []shared.Location{testLocation7}, Implementations: []shared.Location{}},
		{Range: testRange5, HoverText: "text5", Definitions: []shared.Location{testLocation8}, References: nil, Implementations: []shared.Location{}},
		{Range: testRange6, HoverText: "text6", Definitions: []shared.Location{testLocation9}, References: nil, Implementations: []shared.Location{}},
	}

	mockLsifStore.GetRangesFunc.PushReturn(ranges[0:1], nil)
	mockLsifStore.GetRangesFunc.PushReturn(ranges[1:4], nil)
	mockLsifStore.GetRangesFunc.PushReturn(ranges[4:], nil)

	mockRequest := shared.RequestArgs{
		RepositoryID: 42,
		Commit:       mockCommit,
		Path:         mockPath,
		Line:         10,
		Character:    20,
		Limit:        50,
	}
	adjustedRanges, err := svc.GetRanges(context.Background(), mockRequest, mockRequestState, 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying ranges: %s", err)
	}

	adjustedLocation1 := types.UploadLocation{Dump: uploads[0], Path: "sub1/a.go", TargetCommit: "deadbeef", TargetRange: testRange1}
	adjustedLocation2 := types.UploadLocation{Dump: uploads[1], Path: "sub2/b.go", TargetCommit: "deadbeef", TargetRange: testRange2}
	adjustedLocation3 := types.UploadLocation{Dump: uploads[1], Path: "sub2/c.go", TargetCommit: "deadbeef", TargetRange: testRange1}
	adjustedLocation4 := types.UploadLocation{Dump: uploads[1], Path: "sub2/d.go", TargetCommit: "deadbeef", TargetRange: testRange2}
	adjustedLocation5 := types.UploadLocation{Dump: uploads[1], Path: "sub2/e.go", TargetCommit: "deadbeef", TargetRange: testRange1}
	adjustedLocation6 := types.UploadLocation{Dump: uploads[1], Path: "sub2/a.go", TargetCommit: "deadbeef", TargetRange: testRange2}
	adjustedLocation7 := types.UploadLocation{Dump: uploads[1], Path: "sub2/a.go", TargetCommit: "deadbeef", TargetRange: testRange3}
	adjustedLocation8 := types.UploadLocation{Dump: uploads[2], Path: "sub3/a.go", TargetCommit: "deadbeef", TargetRange: testRange4}

	expectedRanges := []shared.AdjustedCodeIntelligenceRange{
		{Range: testRange1, HoverText: "text1", Definitions: []types.UploadLocation{}, References: []types.UploadLocation{adjustedLocation1}, Implementations: []types.UploadLocation{}},
		{Range: testRange2, HoverText: "text2", Definitions: []types.UploadLocation{adjustedLocation2}, References: []types.UploadLocation{adjustedLocation3}, Implementations: []types.UploadLocation{}},
		{Range: testRange3, HoverText: "text3", Definitions: []types.UploadLocation{adjustedLocation4}, References: []types.UploadLocation{adjustedLocation5}, Implementations: []types.UploadLocation{}},
		{Range: testRange4, HoverText: "text4", Definitions: []types.UploadLocation{adjustedLocation6}, References: []types.UploadLocation{adjustedLocation7}, Implementations: []types.UploadLocation{}},
		{Range: testRange5, HoverText: "text5", Definitions: []types.UploadLocation{adjustedLocation8}, References: []types.UploadLocation{}, Implementations: []types.UploadLocation{}},
		// no definition expected, as the line has been changed and we filter those out from range requests
		{Range: testRange6, HoverText: "text6", Definitions: []types.UploadLocation{}, References: []types.UploadLocation{}, Implementations: []types.UploadLocation{}},
	}
	if diff := cmp.Diff(expectedRanges, adjustedRanges); diff != "" {
		t.Errorf("unexpected ranges (-want +got):\n%s", diff)
	}
}
