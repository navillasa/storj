// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package boltdb

import (
	"bytes"
	"os"
	"testing"

	"go.uber.org/zap"
	"storj.io/storj/storage"
)

func TestNetState(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	c, err := NewClient(logger, tempfile(), "test_bucket")
	if err != nil {
		t.Error("Failed to create test db")
	}
	defer func() {
		c.Close()
		switch client := c.(type) {
		case *boltClient:
			os.Remove(client.Path)
		}
	}()

	// tests Put function
	if err := c.Put([]byte("test/path"), []byte("pointer1")); err != nil {
		t.Error("Failed to save testFile to pointers bucket")
	}

	// tests Get function
	retrvValue, err := c.Get([]byte("test/path"))
	if err != nil {
		t.Error("Failed to get saved test pointer")
	}
	if !bytes.Equal(retrvValue, []byte("pointer1")) {
		t.Error("Retrieved pointer was not same as put pointer")
	}

	// // tests Get non-existent path
	_, err = c.Get([]byte("fake test"))
	if err == nil {
		t.Fatal(err)
	}

	// tests Delete function
	if err := c.Delete([]byte("test/path")); err != nil {
		t.Error("Failed to delete test entry")
	}

	// tests List function
	if err := c.Put([]byte("test/path/2"), []byte("pointer2")); err != nil {
		t.Error("Failed to put testEntry2 to pointers bucket")
	}
	testPaths, err := c.List([]byte("test/path/2"), storage.Limit(1))
	if err != nil {
		t.Error("Failed to list Path keys in pointers bucket")
	}

	// tests List + Delete function
	if !bytes.Equal(testPaths[0], []byte("test/path/2")) {
		t.Error("Expected only testEntry2 path in list")
	}
}
