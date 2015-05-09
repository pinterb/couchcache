package main

import (
	"flag"
	"fmt"
	"github.com/couchbaselabs/gocb"
	"log"
)

const (
	MAX_TTL_IN_SEC   = 60 * 60 * 24 * 30
	MAX_SIZE_IN_BYTE = 20 * 1024 * 1024
	MAX_KEY_LENGTH   = 250
)

type couchbaseDatastore gocb.Bucket

func newDatastore() (ds *couchbaseDatastore, err error) {
	url, bucket, pass := parseFlag()

	if c, err := gocb.Connect(url); err == nil {
		if b, err := c.OpenBucket(bucket, pass); err == nil {
			return (*couchbaseDatastore)(b), nil
		}
	}
	return nil, err
}

func parseFlag() (string, string, string) {
	host := flag.String("host", "localhost", "host name (defaults to localhost)")
	port := flag.Int("port", 8091, "port number (defaults to 8091)")
	bucket := flag.String("bucket", "couchcache", "bucket name (defaults to couchcache)")
	pass := flag.String("pass", "password", "password (defaults to password)")

	flag.Parse()

	url := fmt.Sprintf("http://%s:%d", *host, *port)
	log.Println(url)
	return url, *bucket, *pass
}

func (ds *couchbaseDatastore) get(k string) []byte {
	var val []uint8
	if _, err := (*gocb.Bucket)(ds).Get(k, &val); err != nil {
		if err.Error() != "Key not found." {
			log.Println(err)
		}
		return nil
	} else {
		return []byte(val)
	}
}

func (ds *couchbaseDatastore) set(k string, v []byte, ttl int) error {
	if err := ds.validKey(k); err != nil {
		return INVALID_KEY
	}

	if err := ds.validValue(v); err != nil {
		return err
	}

	if ttl > MAX_TTL_IN_SEC {
		ttl = MAX_TTL_IN_SEC
	} else if ttl < 0 {
		ttl = 0
	}

	_, err := (*gocb.Bucket)(ds).Upsert(k, v, uint32(ttl))
	return memdErrorToDatastoreError(err)

}

func (ds *couchbaseDatastore) delete(k string) error {
	if err := ds.validKey(k); err != nil {
		return INVALID_KEY
	}

	_, err := (*gocb.Bucket)(ds).Remove(k, uint64(0))
	return memdErrorToDatastoreError(err)
}

func (ds *couchbaseDatastore) append(k string, v []byte) error {
	if err := ds.validKey(k); err != nil {
		return INVALID_KEY
	}

	if err := ds.validValue(v); err != nil {
		return err
	}

	_, err := (*gocb.Bucket)(ds).Append(k, string(v))
	return memdErrorToDatastoreError(err)

}

func (ds *couchbaseDatastore) validKey(key string) error {
	if len(key) < 1 || len(key) > MAX_KEY_LENGTH {
		return INVALID_KEY
	}
	return nil
}

func (ds *couchbaseDatastore) validValue(v []byte) error {
	if len(v) == 0 {
		log.Println("body is empty")
		return EMPTY_BODY
	}

	if len(v) > MAX_SIZE_IN_BYTE {
		log.Println("body is too large")
		return OVERSIZED_BODY
	}

	return nil
}

func memdErrorToDatastoreError(err error) error {
	if err == nil {
		return nil
	}

	switch err.Error() {
	case "Key not found.":
		return NOT_FOUND_ERROR
	case "The document could not be stored.":
		return NOT_FOUND_ERROR
	case "Document value was too large.":
		return OVERSIZED_BODY
	default:
		log.Println(err)
		return err
	}
}
