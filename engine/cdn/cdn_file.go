package cdn

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
)

func (s *Service) storeFile(ctx context.Context, sig cdn.Signature, reader io.ReadCloser) error {
	var itemType sdk.CDNItemType
	switch {
	case sig.Worker.ArtifactName != "":
		itemType = sdk.CDNTypeItemArtifact
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
		return sdk.WrapError(sdk.ErrConflictData, "cannot upload the same artifact twice")
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

	switch itemType {
	case sdk.CDNTypeItemArtifact:
		// CALL CDS API to CHECK IF WE CAN UPLOAD ARTIFACT
		artiApiRef, _ := it.GetCDNArtifactApiRef()
		if err := s.Client.WorkflowRunResultsCheck(ctx, sig.ProjectKey, sig.WorkflowName, sig.RunNumber, *artiApiRef); err != nil {
			return err
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
	defer tx.Rollback()

	// Check and Clean file with same ref
	if err := s.cleanPreviousFileItem(ctx, tx, sig, itemType, apiRef.ToFilename()); err != nil {
		return err
	}

	// Insert Item
	if err := item.Insert(ctx, s.Mapper, tx, it); err != nil {
		return err
	}

	// Insert Item Unit
	iu.ItemID = iu.Item.ID
	if err := storage.InsertItemUnit(ctx, s.Mapper, tx, iu); err != nil {
		return err
	}

	switch itemType {
	case sdk.CDNTypeItemArtifact:
		// Call CDS to insert workflow run result
		artiResult := sdk.WorkflowRunResultArtifact{
			Name:       apiRef.ToFilename(),
			Size:       it.Size,
			MD5:        it.MD5,
			CDNRefHash: it.APIRefHash,
		}
		bts, err := json.Marshal(artiResult)
		if err != nil {
			return sdk.WithStack(err)
		}
		wrResult := sdk.WorkflowRunResult{
			WorkflowRunID:     sig.RunID,
			WorkflowNodeRunID: sig.NodeRunID,
			WorkflowJobID:     sig.JobID,
			Type:              sdk.WorkflowRunResultTypeArtifact,
			DataRaw:           json.RawMessage(bts),
		}
		if err := s.Client.WorkflowRunResultsAdd(ctx, sig.ProjectKey, sig.WorkflowName, sig.RunNumber, wrResult); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	s.Units.PushInSyncQueue(ctx, it.ID, it.Created)
	return nil
}

func (s *Service) cleanPreviousFileItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, sig cdn.Signature, itemType sdk.CDNItemType, name string) error {
	switch itemType {
	case sdk.CDNTypeItemWorkerCache:
		// Check if item already exist
		existingItem, err := item.LoadFileByProjectAndCacheTag(ctx, s.Mapper, tx, itemType, sig.ProjectKey, name)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			return nil
		}
		existingItem.ToDelete = true
		return item.Update(ctx, s.Mapper, tx, existingItem)
	}
	return nil
}
