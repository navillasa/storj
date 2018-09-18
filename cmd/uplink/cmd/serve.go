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

	"github.com/gabriel-vasile/mimetype"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"storj.io/storj/pkg/paths"
	"storj.io/storj/pkg/process"
	"storj.io/storj/pkg/storage/buckets"
	"storj.io/storj/pkg/utils"
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

type routes struct {
	ctx  context.Context
	args []string
}

func newRoutes(ctx context.Context, args []string) *routes {
	return &routes{ctx: ctx, args: args}
}

func serve(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("No object specified to serve")
	}
	ctx := process.Ctx(cmd)
	ro := newRoutes(ctx, args)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello here you are being served\n")
	})
	fmt.Printf("Now listening on %s\n", *addrFlag)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s", *addrFlag), start(ro)))

	return nil
}

func start(ro *routes) *httprouter.Router {
	router := httprouter.New()
	router.GET("/", ro.serveGet)
	// .POST later, which should add a new file (then website can reupload back into network)
	// .PUT can check that thing exists first then do the post
	// r.URL.Path is how to figure out what bucket/file (opaque string needs to be split)
	return router
}

func (ro *routes) serveGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rr, err := ro.handleTheGrab()
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
func (ro *routes) handleTheGrab() (io.ReadCloser, error) {
	u0, err := utils.ParseURL(ro.args[0])
	if err != nil {
		return nil, err
	}
	bs, err := cfg.BucketStore(ro.ctx)
	if err != nil {
		return nil, err
	}
	if u0.Host == "" {
		return nil, fmt.Errorf("No bucket specified. Please use format sj://bucket/")
	}
	r, err := ro.grabObj(ro.ctx, bs, u0)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// grabObj returns s3 compatible object at args[0]
func (ro *routes) grabObj(ctx context.Context, bs buckets.Store, srcObj *url.URL) (io.ReadCloser, error) {
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
