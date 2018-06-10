// Copyright 2018 The go-hep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client // import "go-hep.org/x/hep/xrootd/client"

import (
	"context"

	"go-hep.org/x/hep/xrootd/xrdfs"
	"go-hep.org/x/hep/xrootd/xrdproto/chmod"
	"go-hep.org/x/hep/xrootd/xrdproto/dirlist"
	"go-hep.org/x/hep/xrootd/xrdproto/mkdir"
	"go-hep.org/x/hep/xrootd/xrdproto/mv"
	"go-hep.org/x/hep/xrootd/xrdproto/open"
	"go-hep.org/x/hep/xrootd/xrdproto/rm"
	"go-hep.org/x/hep/xrootd/xrdproto/rmdir"
	"go-hep.org/x/hep/xrootd/xrdproto/stat"
	"go-hep.org/x/hep/xrootd/xrdproto/truncate"
)

// FS returns a xrdfs.FileSystem which uses this client to make requests.
func (cli *Client) FS() xrdfs.FileSystem {
	return &fileSystem{cli}
}

// fileSystem contains filesystem-related methods of the XRootD protocol.
type fileSystem struct {
	c *Client
}

// Dirlist returns the contents of a directory together with the stat information.
func (fs *fileSystem) Dirlist(ctx context.Context, path string) ([]xrdfs.EntryStat, error) {
	var resp dirlist.Response
	err := fs.c.Send(ctx, &resp, dirlist.NewRequest(path))
	if err != nil {
		return nil, err
	}
	return resp.Entries, err
}

// Open returns the file handle for a file together with the compression and the stat info.
func (fs *fileSystem) Open(ctx context.Context, path string, mode xrdfs.OpenMode, options xrdfs.OpenOptions) (xrdfs.File, error) {
	var resp open.Response
	err := fs.c.Send(ctx, &resp, open.NewRequest(path, mode, options))
	if err != nil {
		return nil, err
	}
	return &file{fs, resp.FileHandle, resp.Compression, resp.Stat}, nil
}

// RemoveFile removes a file.
func (fs *fileSystem) RemoveFile(ctx context.Context, path string) error {
	_, err := fs.c.call(ctx, &rm.Request{Path: path})
	return err
}

// Truncate changes the size of the named file.
func (fs *fileSystem) Truncate(ctx context.Context, path string, size int64) error {
	_, err := fs.c.call(ctx, &truncate.Request{Path: path, Size: size})
	return err
}

// Stat returns the entry stat info for the given path.
func (fs *fileSystem) Stat(ctx context.Context, path string) (xrdfs.EntryStat, error) {
	var resp stat.DefaultResponse
	err := fs.c.Send(ctx, &resp, &stat.Request{Path: path})
	if err != nil {
		return xrdfs.EntryStat{}, err
	}
	return resp.EntryStat, nil
}

// VirtualStat returns the virtual filesystem stat info for the given path.
// Note that path needs not to be an existing filesystem object, it is used as a path prefix in order to
// filter out servers and partitions that could not be used to hold objects whose path starts
// with the specified path prefix.
func (fs *fileSystem) VirtualStat(ctx context.Context, path string) (xrdfs.VirtualFSStat, error) {
	var resp stat.VirtualFSResponse
	err := fs.c.Send(ctx, &resp, &stat.Request{Path: path, Options: stat.OptionsVFS})
	if err != nil {
		return xrdfs.VirtualFSStat{}, err
	}
	return resp.VirtualFSStat, nil
}

func (fs *fileSystem) Mkdir(ctx context.Context, path string, perm xrdfs.OpenMode) error {
	_, err := fs.c.call(ctx, &mkdir.Request{Path: path, Mode: perm})
	return err
}

func (fs *fileSystem) MkdirAll(ctx context.Context, path string, perm xrdfs.OpenMode) error {
	_, err := fs.c.call(ctx, &mkdir.Request{Path: path, Mode: perm, Options: mkdir.OptionsMakePath})
	return err
}

// RemoveDir removes a directory.
// The directory to be removed must be empty.
func (fs *fileSystem) RemoveDir(ctx context.Context, path string) error {
	_, err := fs.c.call(ctx, &rmdir.Request{Path: path})
	return err
}

// Rename renames (moves) oldpath to newpath.
func (fs *fileSystem) Rename(ctx context.Context, oldpath, newpath string) error {
	_, err := fs.c.call(ctx, &mv.Request{OldPath: oldpath, NewPath: newpath})
	return err
}

// Chmod changes the permissions of the named file to perm.
func (fs *fileSystem) Chmod(ctx context.Context, path string, perm xrdfs.OpenMode) error {
	_, err := fs.c.call(ctx, &chmod.Request{Path: path, Mode: perm})
	return err
}

var (
	_ xrdfs.FileSystem = (*fileSystem)(nil)
)
