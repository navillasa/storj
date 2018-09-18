// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"storj.io/storj/pkg/paths"
	"storj.io/storj/pkg/storage/buckets"
)

var (
	addrFlag *string
)

func init() {
	serveCmd := addCmd(&cobra.Command{
		Use:   "serve",
		Short: "Handles any requests and writes response to browser",
		RunE:  serve,
	})
	addrFlag = serveCmd.Flags().String("addr", ":8080", "address to listen on")
}

func serve(cmd *cobra.Command, args []string) error {
	http.HandleFunc("/", serveGet)
	fmt.Printf("Now listening on %s\n", *addrFlag)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s", *addrFlag), nil))

	return nil
}

func serveGet(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.SplitN(r.URL.Path, "/", 3)
	newURL := &url.URL{
		Host:   pathParts[1],
		Scheme: "sj",
		Path:   pathParts[2],
	}

	rr, err := handleTheGrab(newURL)
	//first arg in url path needs to be set as host, everything else is the path
	if err != nil {
		http.Error(w, "bad request: err grabbing response", http.StatusBadRequest)
		log.Fatal("err grabbing response", err)
		return
	}

	b, err := ioutil.ReadAll(rr)
	if err != nil {
		http.Error(w, "bad request: err reading response", http.StatusBadRequest)
		log.Fatal("err reading response", err)
	}

	mime, _ := mimetype.Detect(b)
	if err != nil {
		http.Error(w, "bad request: err grabbing response", http.StatusBadRequest)
		log.Fatal("err grabbing response", err)
	}
	w.Header().Set("Content-Type", mime)

	_, err = w.Write(b)
	if err != nil {
		log.Fatal("err writing response", zap.Error(err))
	}
}

// handleTheGrab gets ready to grab
func handleTheGrab(src *url.URL) (io.ReadCloser, error) {
	ctx := context.Background()
	bs, err := cfg.BucketStore(ctx)
	if err != nil {
		return nil, err
	}
	if src.Host == "" {
		return nil, fmt.Errorf("No bucket specified. Please use format sj://bucket/")
	}
	r, err := grabObj(ctx, bs, src)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// grabObj returns s3 compatible object at args[0]
func grabObj(ctx context.Context, bs buckets.Store, srcObj *url.URL) (io.ReadCloser, error) {
	if srcObj.Scheme == "" {
		return nil, fmt.Errorf("Invalid source")
	}
	o, err := bs.GetObjectStore(ctx, srcObj.Host)
	if err != nil {
		return nil, err
	}
	rr, _, err := o.Get(ctx, paths.New(srcObj.Path))
	if err != nil {
		return nil, err
	}
	r, err := rr.Range(ctx, 0, rr.Size())
	if err != nil {
		return nil, err
	}
	return r, nil
}
