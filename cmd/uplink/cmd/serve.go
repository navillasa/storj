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
	"time"

	"github.com/gabriel-vasile/mimetype"
	minio "github.com/minio/minio/cmd"
	"github.com/spf13/cobra"

	"storj.io/storj/pkg/paths"
	"storj.io/storj/pkg/storage/meta"
	"storj.io/storj/pkg/storage/objects"
	"storj.io/storj/storage"
)

var (
	addrFlag *string
)

func init() {
	serveCmd := addCmd(&cobra.Command{
		Use:   "serve",
		Short: "Serves objects to the browser",
		RunE:  serve,
	})
	addrFlag = serveCmd.Flags().String("addr", ":8080", "address to listen on")
}

func serve(cmd *cobra.Command, args []string) error {
	http.HandleFunc("/", serveEverything)
	fmt.Printf("Now listening on %s\n", *addrFlag)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s", *addrFlag), nil))

	return nil
}

func serveEverything(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()

		pathParts := strings.SplitN(r.URL.Path, "/", 3)
		if len(pathParts) != 3 {
			http.Error(w, "Error parsing bucket and path. Make sure to use format localhost:8080/bucket/path.", http.StatusBadRequest)
			log.Println("Error parsing bucket and path. Make sure to use format localhost:8080/bucket/path.")
			return
		}
		log.Printf("%#v\n", pathParts)

		destObj := &url.URL{
			Scheme: "sj",
			Host:   pathParts[1],
			Path:   handler.Filename,
		}
		log.Printf("%#v\n", destObj.Path)

		ctx := context.Background()
		bs, err := cfg.BucketStore(ctx)
		if err != nil {
			return
		}

		o, err := bs.GetObjectStore(ctx, destObj.Host)
		if err != nil {
			return
		}

		meta := objects.SerializableMeta{}
		expTime := time.Time{}

		_, err = o.Put(ctx, paths.New(destObj.Path), file, meta, expTime)
		if err != nil {
			return
		}
	} else {
		pathParts := strings.SplitN(r.URL.Path, "/", 3)
		if len(pathParts) == 2 {
			ctx := context.Background()
			bs, err := cfg.BucketStore(ctx)
			if err != nil {
				return
			}
			src := &url.URL{
				Scheme: "sj",
				Host:   pathParts[1],
			}
			o, err := bs.GetObjectStore(ctx, src.Host)
			if err != nil {
				return
			}
			startAfter := paths.New("")
			for {
				items, more, err := o.List(ctx, paths.New(src.Path), startAfter, nil, false, 0, meta.Modified|meta.Size)
				if err != nil {
					return
				}
				var writeMe []string
				for _, object := range items {
					path := object.Path.String()
					if object.IsPrefix {
						fmt.Println("PRE", path+"/")
					} else {
						writeMe = append(writeMe, fmt.Sprintf("%v\n", path))
					}
				}
				fmt.Fprintf(w, strings.Join(writeMe, ""))
				if !more {
					break
				}
				startAfter = items[len(items)-1].Path
			}
			return
		}
		if len(pathParts) == 3 {
			newURL := &url.URL{
				Scheme: "sj",
				Host:   pathParts[1],
				Path:   pathParts[2],
			}
			rr, err := grabObj(newURL)
			if err != nil {
				if storage.ErrKeyNotFound.Has(err) {
					http.Error(w, "404 Object not found: "+r.URL.Path, http.StatusNotFound)
					log.Println("Error object not found: " + r.URL.Path)
					return
				}
				if (err == minio.BucketNotFound{Bucket: pathParts[1]}) {
					http.Error(w, "404 "+err.Error(), http.StatusNotFound)
					log.Println(err)
					return
				}
				http.Error(w, "500 error downloading object", http.StatusInternalServerError)
				log.Println("Error downloading object: ", err)
				return
			}
			b, err := ioutil.ReadAll(rr)
			if err != nil {
				http.Error(w, "500 error reading object's reader", http.StatusInternalServerError)
				log.Fatal("Error reading object's reader", err)
				return
			}
			mime, _ := mimetype.Detect(b)
			w.Header().Set("Content-Type", mime)
			_, err = w.Write(b)
			if err != nil {
				http.Error(w, "500 error writing object", http.StatusInternalServerError)
				log.Fatal("Error writing object", err)
				return
			}
		}
	}
}

// grabObj returns a S3 compatible object
func grabObj(src *url.URL) (io.ReadCloser, error) {
	ctx := context.Background()
	bs, err := cfg.BucketStore(ctx)
	if err != nil {
		return nil, err
	}
	if src.Host == "" {
		return nil, fmt.Errorf("no bucket specified: please use format localhost:8080/bucket/path")
	}
	o, err := bs.GetObjectStore(ctx, src.Host)
	if err != nil {
		return nil, err
	}
	rr, _, err := o.Get(ctx, paths.New(src.Path))
	if err != nil {
		return nil, err
	}
	r, err := rr.Range(ctx, 0, rr.Size())
	if err != nil {
		return nil, err
	}
	return r, nil
}
