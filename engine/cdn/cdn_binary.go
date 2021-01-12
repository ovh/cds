package cdn

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
)

func (s *Service) storeFile(ctx context.Context, sig cdn.Signature, reader io.ReadCloser) error {
	var itemType sdk.CDNItemType
	switch {
	case sig.Worker.ArtifactName != "":
		itemType = sdk.CDNTypeItemArtifact
	default:
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid item type")
	}
	bufferUnit := s.Units.FileBuffer()

	// Item and ItemUnit creation
	apiRef := sdk.NewCDNApiRef(itemType, sig)
	hashRef, err := apiRef.ToHash()
	if err != nil {
		return err
	}
	it := &sdk.CDNItem{
		APIRef:     apiRef,
		Type:       itemType,
		APIRefHash: hashRef,
		Status:     sdk.CDNStatusItemIncoming,
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
	// For optimum speed, Getpagesize returns the underlying system's memory page size.
	pagesize := os.Getpagesize()
	// wraps the Reader object into a new buffered reader to read the files in chunks
	// and buffering them for performance.
	mreader := bufio.NewReaderSize(reader, pagesize)
	multiWriter := io.MultiWriter(md5Hash, sha512Hash, writer)

	size, err := io.Copy(multiWriter, mreader)
	if err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return sdk.WithStack(err)
	}
	if err := reader.Close(); err != nil {
		return sdk.WithStack(err)
	}
	if err := writer.Close(); err != nil {
		return err
	}

	sha512S := hex.EncodeToString(sha512Hash.Sum(nil))
	md5S := hex.EncodeToString(md5Hash.Sum(nil))

	it.Hash = sha512S
	it.MD5 = md5S
	it.Size = size
	it.Status = sdk.CDNStatusItemCompleted

	// Insert Item and ItemUnit in database
	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WithStack(err)
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

	return sdk.WithStack(tx.Commit())
}
