package cdn

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
)

type StoreFileOptions struct {
	DisableApiRunResult bool
}

func (s *Service) storeFile(ctx context.Context, sig cdn.Signature, reader io.ReadCloser, storeFileOptions StoreFileOptions) error {
	var itemType sdk.CDNItemType
	switch {
	case sig.Worker.FileName != "":
		itemType = sdk.CDNTypeItemRunResult
	case sig.Worker.CacheTag != "":
		itemType = sdk.CDNTypeItemWorkerCache
	default:
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid item type")
	}
	bufferUnit := s.Units.FileBuffer()

	// Item and ItemUnit creation
	apiRef, err := sdk.NewCDNApiRef(itemType, sig)
	if err != nil {
		return err
	}
	hashRef, err := apiRef.ToHash()
	if err != nil {
		return err
	}

	// Check Item unicity
	_, err = item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), hashRef, itemType)
	if err == nil {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "cannot upload the same file twice: %s", apiRef.ToFilename())
	}
	if !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	it := &sdk.CDNItem{
		APIRef:     apiRef,
		Type:       itemType,
		APIRefHash: hashRef,
		Status:     sdk.CDNStatusItemIncoming,
	}

	if !storeFileOptions.DisableApiRunResult {
		switch itemType {
		case sdk.CDNTypeItemRunResult:
			// Call CDS API to check if we can upload the run result
			runResultApiRef, _ := it.GetCDNRunResultApiRef()

			runResultCheck := sdk.WorkflowRunResultCheck{
				Name:       runResultApiRef.ArtifactName,
				ResultType: runResultApiRef.RunResultType,
				RunID:      runResultApiRef.RunID,
				RunNodeID:  runResultApiRef.RunNodeID,
				RunJobID:   runResultApiRef.RunJobID,
			}
			code, err := s.Client.QueueWorkflowRunResultCheck(ctx, sig.JobID, runResultCheck)
			if err != nil {
				if code == http.StatusConflict {
					return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to upload the same file twice: %s", runResultApiRef.ToFilename())
				}
				return err
			}
		}
	}

	iu, err := s.Units.NewItemUnit(ctx, bufferUnit, it)
	if err != nil {
		return err
	}

	// Create Destination Writer
	writer, err := bufferUnit.NewWriter(ctx, *iu)
	if err != nil {
		return err
	}

	// Compute md5 and sha512
	md5Hash := md5.New()
	sha512Hash := sha512.New()

	sizeWriter := &SizeWriter{}

	// For optimum speed, Getpagesize returns the underlying system's memory page size.
	pagesize := os.Getpagesize()
	// wraps the Reader object into a new buffered reader to read the files in chunks
	// and buffering them for performance.
	mreader := bufio.NewReaderSize(reader, pagesize)
	multiWriter := io.MultiWriter(md5Hash, sha512Hash, sizeWriter)

	teeReader := io.TeeReader(mreader, multiWriter)

	if err := bufferUnit.Write(*iu, teeReader, writer); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return sdk.WithStack(err)
	}
	if err := reader.Close(); err != nil {
		return sdk.WithStack(err)
	}
	sha512S := hex.EncodeToString(sha512Hash.Sum(nil))
	md5S := hex.EncodeToString(md5Hash.Sum(nil))

	it.Hash = sha512S
	it.MD5 = md5S
	it.Size = sizeWriter.Size
	it.Status = sdk.CDNStatusItemCompleted

	// Insert Item and ItemUnit in database
	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	// Insert Item
	if err := item.Insert(ctx, s.Mapper, tx, it); err != nil {
		return err
	}

	// Insert Item Unit
	iu.ItemID = iu.Item.ID
	if err := storage.InsertItemUnit(ctx, s.Mapper, tx, iu); err != nil {
		return err
	}

	if !storeFileOptions.DisableApiRunResult {
		runResultApiRef, _ := it.GetCDNRunResultApiRef()
		switch itemType {
		case sdk.CDNTypeItemRunResult:
			var result interface{}
			switch runResultApiRef.RunResultType {
			case sdk.WorkflowRunResultTypeArtifact:
				result = sdk.WorkflowRunResultArtifact{
					WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
						Name: apiRef.ToFilename(),
					},
					Size:       it.Size,
					MD5:        it.MD5,
					CDNRefHash: it.APIRefHash,
					Perm:       runResultApiRef.Perm,
				}
			case sdk.WorkflowRunResultTypeCoverage:
				result = sdk.WorkflowRunResultCoverage{
					WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
						Name: apiRef.ToFilename(),
					},
					Size:       it.Size,
					MD5:        it.MD5,
					CDNRefHash: it.APIRefHash,
					Perm:       runResultApiRef.Perm,
				}
			}

			bts, err := json.Marshal(result)
			if err != nil {
				return sdk.WithStack(err)
			}
			wrResult := sdk.WorkflowRunResult{
				WorkflowRunID:     sig.RunID,
				WorkflowNodeRunID: sig.NodeRunID,
				WorkflowRunJobID:  sig.JobID,
				Type:              runResultApiRef.RunResultType,
				DataRaw:           json.RawMessage(bts),
			}
			if err := s.Client.QueueWorkflowRunResultsAdd(ctx, sig.JobID, wrResult); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	s.Units.PushInSyncQueue(ctx, it.ID, it.Created)

	// For worker cache item clean others with same ref to purge old cached data
	if itemType == sdk.CDNTypeItemWorkerCache {
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() //nolint

		if err := s.cleanPreviousCachedData(ctx, tx, sig, apiRef.ToFilename()); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
	}

	return nil
}

// Mark to delete all items for given cache tag except the most recent one.
func (s *Service) cleanPreviousCachedData(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, sig cdn.Signature, cacheTag string) error {
	items, err := item.LoadWorkerCacheItemsByProjectAndCacheTag(ctx, s.Mapper, tx, sig.ProjectKey, cacheTag)
	if err != nil {
		return err
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Created.Before(items[j].Created) })

	for i := 0; i < len(items)-1; i++ {
		items[i].ToDelete = true
		if err := item.Update(ctx, s.Mapper, tx, &items[i]); err != nil {
			return err
		}
	}

	return nil
}
