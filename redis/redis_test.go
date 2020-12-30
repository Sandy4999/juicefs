/*
 * JuiceFS, Copyright (C) 2020 Juicedata, Inc.
 *
 * This program is free software: you can use, redistribute, and/or modify
 * it under the terms of the GNU Affero General Public License, version 3
 * or later ("AGPL"), as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
 * FITNESS FOR A PARTICULAR PURPOSE.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package redis

import (
	"testing"

	"github.com/juicedata/juicefs/meta"
)

func TestRedisClient(t *testing.T) {
	var conf RedisConfig
	m, err := NewRedisMeta("redis://127.0.0.1:6379/11", &conf)
	if err != nil {
		t.Logf("redis is not available: %s", err)
		t.Skip()
	}
	m.OnMsg(meta.CHUNK_DEL, func(args ...interface{}) error { return nil })
	ctx := meta.Background
	var parent, inode meta.Ino
	var attr = &meta.Attr{}
	m.GetAttr(ctx, 1, 0, attr) // init
	if st := m.Mkdir(ctx, 1, "d", 0640, 022, 0, &parent, attr); st != 0 {
		t.Fatalf("mkdir %s", st)
	}
	if st := m.Lookup(ctx, 1, "d", &parent, attr); st != 0 {
		t.Fatalf("lookup dir: %s", st)
	}
	if st := m.Create(ctx, parent, "f", 0650, 022, &inode, attr); st != 0 {
		t.Fatalf("create file %s", st)
	}
	if st := m.Lookup(ctx, parent, "f", &inode, attr); st != 0 {
		t.Fatalf("lookup file: %s", st)
	}
	attr.Mtime = 2
	attr.Uid = 1
	if st := m.SetAttr(ctx, inode, 0, meta.SET_ATTR_MTIME|meta.SET_ATTR_UID, 0, attr); st != 0 {
		t.Fatalf("setattr file %s", st)
	}
	if st := m.GetAttr(ctx, inode, 0, attr); st != 0 {
		t.Fatalf("getattr file %s", st)
	}
	if attr.Mtime != 2 || attr.Uid != 1 {
		t.Fatalf("mtime:%d uid:%d", attr.Mtime, attr.Uid)
	}
	var entries []*meta.Entry
	if st := m.Readdir(ctx, parent, 0, &entries); st != 0 {
		t.Fatalf("readdir: %s", st)
	} else if len(entries) != 1 {
		t.Fatalf("entries: %d", len(entries))
	}
	if st := m.Rename(ctx, parent, "f", 1, "f2", &inode, attr); st != 0 {
		t.Fatalf("rename f %s", st)
	}
	if st := m.Lookup(ctx, 1, "f2", &inode, attr); st != 0 {
		t.Fatalf("lookup f2: %s", st)
	}

	// data
	var chunkid uint64
	if st := m.Open(ctx, inode, 2, attr); st != 0 {
		t.Fatalf("open f2: %s", st)
	}
	if st := m.NewChunk(ctx, inode, 0, 0, &chunkid); st != 0 {
		t.Fatalf("write chunk: %s", st)
	}
	var s = meta.Slice{chunkid, 100, 0, 100}
	if st := m.Write(ctx, inode, 0, 100, s); st != 0 {
		t.Fatalf("write end: %s", st)
	}
	var chunks []meta.Slice
	if st := m.Read(inode, 0, &chunks); st != 0 {
		t.Fatalf("read chunk: %s", st)
	}
	if len(chunks) != 1 || chunks[0].Chunkid != chunkid || chunks[0].Clen != 100 {
		t.Fatalf("chunks: %v", chunks)
	}

	if st := m.Unlink(ctx, 1, "f2"); st != 0 {
		t.Fatalf("unlink: %s", st)
	}
	if st := m.Rmdir(ctx, 1, "d"); st != 0 {
		t.Fatalf("rmdir: %s", st)
	}
}