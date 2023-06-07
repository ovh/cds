package nfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	gonfs "github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type Buffer struct {
	storage.AbstractUnit
	encryption.NoConvergentEncryption
	config     storage.NFSBufferConfiguration
	bufferType storage.CDNBufferType
	size       int64
	pingStatus string
}

var (
	_ storage.FileBufferUnit = new(Buffer)
)

const driverBufferName = "nfs-buffer"

func init() {
	storage.RegisterDriver(driverBufferName, new(Buffer))
}

func (n *Buffer) GetDriverName() string {
	return driverBufferName
}

type Reader struct {
	ctx       context.Context
	dialMount *gonfs.Mount
	target    *gonfs.Target
	reader    io.ReadCloser
}

func (r *Reader) Close() error {
	var firstError error
	if err := r.reader.Close(); err != nil {
		firstError = err
		log.Error(r.ctx, "reader: unable to close file: %v", err)
	}
	if err := r.target.Close(); err != nil {
		log.Error(r.ctx, "reader: unable to close mount: %v", err)
		if firstError == nil {
			firstError = err
		}
	}
	if err := r.dialMount.Close(); err != nil {
		log.Error(r.ctx, "reader: unable to close DialMount: %v", err)
		if firstError == nil {
			firstError = err
		}
	}
	return sdk.WithStack(firstError)
}

func (r *Reader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

type Writer struct {
	ctx       context.Context
	dialMount *gonfs.Mount
	target    *gonfs.Target
	writer    io.WriteCloser
}

func (w *Writer) Close() error {
	var firstError error
	if err := w.writer.Close(); err != nil {
		firstError = err
		log.Error(w.ctx, "writer: unable to close file: %v", err)
	}
	if err := w.target.Close(); err != nil {
		log.Error(w.ctx, "writer: unable to close mount: %v", err)
		if firstError == nil {
			firstError = err
		}
	}
	if err := w.dialMount.Close(); err != nil {
		log.Error(w.ctx, "writer: unable to close DialMount: %v", err)
		if firstError == nil {
			firstError = err
		}
	}
	return sdk.WithStack(firstError)
}

func (w *Writer) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

func (n *Buffer) Connect() (*gonfs.Mount, *gonfs.Target, error) {
	dialMount, err := gonfs.DialMount(n.config.Host)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "unable to dial mount")
	}
	hostname, err := os.Hostname()
	if err != nil {
		_ = dialMount.Close()
		return nil, nil, sdk.WrapError(err, "unable to get hostname")
	}
	auth := rpc.NewAuthUnix(hostname, n.config.UserID, n.config.GroupID)
	v, err := dialMount.Mount(n.config.TargetPartition, auth.Auth())
	if err != nil {
		_ = dialMount.Close()
		return nil, nil, sdk.WrapError(err, "unable to mount volume %s", n.config.TargetPartition)
	}
	return dialMount, v, nil
}

func (n *Buffer) Init(ctx context.Context, cfg interface{}, bufferType storage.CDNBufferType) error {
	config, is := cfg.(*storage.NFSBufferConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	n.config = *config
	n.NoConvergentEncryption = encryption.NewNoConvergentEncryption(config.Encryption)
	n.bufferType = bufferType

	// Test connection
	d, v, err := n.Connect()
	if err != nil {
		return err
	}
	defer d.Close()
	defer v.Close()

	n.GoRoutines.Run(ctx, "cdn-nfs-buffer-compute-size", func(ctx context.Context) {
		n.computeSize(ctx)
	})

	return nil
}

func (n *Buffer) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	dial, target, err := n.Connect()
	if err != nil {
		return false, err
	}
	defer dial.Close()   //nolint
	defer target.Close() //nolint

	iu, err := n.ExistsInDatabase(ctx, m, db, i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	// Lookup on the filesystem according to the locator
	path, err := n.filename(target, *iu)
	if err != nil {
		return false, err
	}
	_, _, err = target.Lookup(path)
	if err != nil {
		return false, sdk.WithStack(err)
	}
	return true, nil
}

func (n *Buffer) Status(_ context.Context) []sdk.MonitoringStatusLine {
	return []sdk.MonitoringStatusLine{
		{
			Component: fmt.Sprintf("storage/%s/ping", n.Name()),
			Value:     "connect OK",
			Status:    n.pingStatus,
		},
		{
			Component: fmt.Sprintf("storage/%s/size", n.Name()),
			Value:     fmt.Sprintf("%d octets", n.size),
			Status:    sdk.MonitoringStatusOK,
		}}
}

func (n *Buffer) Remove(ctx context.Context, i sdk.CDNItemUnit) error {
	dial, target, err := n.Connect()
	if err != nil {
		return err
	}
	defer dial.Close()   //nolint
	defer target.Close() //nolint

	path, err := n.filename(target, i)
	if err != nil {
		return err
	}
	log.Debug(ctx, "[%T] remove %s", n, path)
	if err := target.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return sdk.ErrNotFound
		}
		return sdk.WithStack(err)
	}
	return nil
}

func (n *Buffer) NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	dial, target, err := n.Connect()
	if err != nil {
		return nil, err
	}

	// Open the file from the filesystem according to the locator
	path, err := n.filename(target, i)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	log.Debug(ctx, "[%T] reading from %s", n, path)
	f, err := target.Open(path)

	nfsReader := &Reader{ctx: ctx, dialMount: dial, target: target, reader: f}

	return nfsReader, sdk.WithStack(err)
}

func (n *Buffer) NewWriter(ctx context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error) {
	dial, target, err := n.Connect()
	if err != nil {
		return nil, err
	}

	// Open the file from the filesystem according to the locator
	path, err := n.filename(target, i)
	if err != nil {
		return nil, err
	}
	log.Debug(ctx, "[%T] writing to %s", n, path)

	f, err := target.OpenFile(path, os.FileMode(0640))
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	return &Writer{ctx: ctx, dialMount: dial, target: target, writer: f}, nil
}

func (n *Buffer) filename(target *gonfs.Target, i sdk.CDNItemUnit) (string, error) {
	if _, err := target.Mkdir(filepath.Join(string(i.Type)), os.FileMode(0775)); err != nil {
		if !os.IsExist(err) {
			return "", sdk.WithStack(err)
		}
	}
	return filepath.Join(string(i.Type), i.Item.APIRefHash), nil
}

func (n *Buffer) Size(_ sdk.CDNItemUnit) (int64, error) {
	return n.size, nil
}

func (n *Buffer) BufferType() storage.CDNBufferType {
	return n.bufferType
}

func (n *Buffer) computeSize(ctx context.Context) {
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:backend:nfs:buffer:computeSize: %v", ctx.Err())
			}
			return
		case <-tick.C:
			s, err := n.dirSize()
			if err != nil {
				log.Error(ctx, "cdn:backend:nfs:buffer:computeSize: unable to compute size: %v", ctx.Err())
			}
			n.size = s
		}
	}
}

func (n *Buffer) dirSize() (int64, error) {
	dial, target, err := n.Connect()
	if err != nil {
		n.pingStatus = sdk.MonitoringStatusAlert
		return 0, err
	}
	defer dial.Close()   // nolint
	defer target.Close() //
	n.pingStatus = sdk.MonitoringStatusOK

	return n.computeDirSizeRecursive(target, ".")
}

func (n *Buffer) computeDirSizeRecursive(v *gonfs.Target, path string) (int64, error) {
	size := int64(0)
	entries, err := n.ls(v, path)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.FileName, ".") {
			continue
		}
		if e.IsDir() {
			sizeR, err := n.computeDirSizeRecursive(v, filepath.Join(path, e.FileName))
			if err != nil {
				return 0, err
			}
			size += sizeR
			continue
		}
		size += e.Size()
	}
	return size, nil
}

func (n *Buffer) ls(v *gonfs.Target, path string) ([]*gonfs.EntryPlus, error) {
	dirs, err := v.ReadDirPlus(path)
	if err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrUnknownError, "readdir error: %v", err)
	}
	return dirs, nil
}

func (n *Buffer) ResyncWithDatabase(ctx context.Context, db gorp.SqlExecutor, t sdk.CDNItemType, dryRun bool) {
	dial, target, err := n.Connect()
	if err != nil {
		log.Error(ctx, "nfs-buffer: unable to connect to NFS: %v", err)
		return
	}
	defer dial.Close()   // nolint
	defer target.Close() //

	entries, err := n.ls(target, string(t))
	if err != nil {
		log.Error(ctx, "nfs-buffer: unable to list directory %s", string(t))
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			log.Warn(ctx, "nfs-buffer: found directory inside %s: %s", string(t), e.FileName)
			continue
		}
		if e.FileName == "" {
			log.Warn(ctx, "nfs-buffer: missing file name")
			continue
		}
		log.Info(ctx, "Found file %s [%d]", e.FileName, e.Size())
		has, err := storage.HashItemUnitByApiRefHash(db, e.FileName, n.ID())
		if err != nil {
			log.Error(ctx, "nfs-buffer: unable to check if unit item exist for api ref hash %s: %v", e.FileName, err)
			continue
		}
		if has {
			continue
		}
		if !dryRun {
			if err := target.Remove(string(t) + "/" + e.FileName); err != nil {
				log.Error(ctx, "nfs-buffer: unable to remove file %s: %v", string(t)+"/"+e.FileName, err)
				continue
			}
			log.Info(ctx, "nfs-buffer: file %s has been deleted", e.FileName)
		} else {
			log.Info(ctx, "nfs-buffer: file %s should be deleted", e.FileName)
		}
	}
	return
}
