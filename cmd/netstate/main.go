// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"flag"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	monkit "gopkg.in/spacemonkeygo/monkit.v2"

	"storj.io/storj/pkg/netstate"
	"storj.io/storj/pkg/process"
	proto "storj.io/storj/protos/netstate"
	"storj.io/storj/storage/boltdb"
)

var (
	port         = flag.Int("port", 8080, "port")
	dbPath       = flag.String("db", "netstate.db", "db path")
	tlsCertPath  = flag.String("tlsCertPath", "", "TLS Certificate file")
	tlsKeyPath   = flag.String("tlsKeyPath", "", "TLS Key file")
	tlsHosts     = flag.String("tlsHosts", "", "TLS Hosts file")
	tlsCreate    = flag.Bool("tlsCreate", false, "If true, generate a new TLS cert/key files")
	tlsOverwrite = flag.Bool("tlsOverwrite", false, "If true, overwrite existing TLS cert/key files")
)

func (s *serv) Process(ctx context.Context) error {
	bdb, err := boltdb.New(s.logger, *dbPath)
	if err != nil {
		return err
	}
	defer bdb.Close()

	// start grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	proto.RegisterNetStateServer(grpcServer, netstate.NewServer(bdb, s.logger))
	s.logger.Debug(fmt.Sprintf("server listening on port %d", *port))

	defer grpcServer.GracefulStop()
	return grpcServer.Serve(lis)
}

type serv struct {
	logger  *zap.Logger
	metrics *monkit.Registry
}

func (s *serv) SetLogger(l *zap.Logger) error {
	s.logger = l
	return nil
}

func (s *serv) SetMetricHandler(m *monkit.Registry) error {
	s.metrics = m
	return nil
}

func (s *serv) InstanceID() string { return "" }

func main() {
	process.Must(process.Main(&serv{}))
}
