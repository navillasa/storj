// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package netstate

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	proto "storj.io/storj/protos/netstate"
)

func TestNetStateClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9000))
	assert.NoError(t, err)

	mdb := &mockDB{
		timesCalled: 0,
	}

	grpcServer := grpc.NewServer()
	proto.RegisterNetStateServer(grpcServer, NewServer(mdb, logger))

	defer grpcServer.GracefulStop()
	go grpcServer.Serve(lis)

	address := lis.Addr().String()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	assert.NoError(t, err)

	c := proto.NewNetStateClient(conn)

	ctx := context.Background()

	// example file path to put/get
	fp := proto.FilePath{
		Path:       "here/is/a/path",
		SmallValue: "oatmeal",
	}

	if mdb.timesCalled != 0 {
		t.Error("Expected mockdb to be called 0 times")
	}

	// Tests NetState.Put
	res, err := c.Put(ctx, &fp)
	assert.NoError(t, err)

	if res.Confirmation != "success" {
		t.Error("Failed to receive success Put response")
	}

	if mdb.timesCalled != 1 {
		t.Error("Failed to call mockdb correctly")
	}

	if mdb.puts[0].Path != fp.Path || string(mdb.puts[0].Value) != fp.SmallValue {
		t.Error("Expected path to be in mockdb")
	}
}
