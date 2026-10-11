package storage

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"sort"
)

type Descriptor struct {
	Kind string    `json:"kind"`
	Root string    `json:"root,omitempty"`
	S3   *S3Config `json:"s3,omitempty"`
}

func (d Descriptor) Open() (Provider, error) {
	if d.Kind == "local" && d.S3 == nil {
		return Local{Root: d.Root}, nil
	}
	if d.Kind == "s3" && d.S3 != nil {
		return NewS3(*d.S3)
	}
	return nil, errors.New("invalid provider descriptor")
}
func (r Registry) Describe(ctx context.Context, id string) (Descriptor, error) {
	var d Descriptor
	var raw, box []byte
	var keyID *string
	if e := r.DB.QueryRow(ctx, "SELECT kind,config,secret_key_id::text,secret_envelope FROM storage_providers WHERE id=$1", id).Scan(&d.Kind, &raw, &keyID, &box); e != nil {
		return d, e
	}
	if d.Kind == "local" {
		var v struct {
			Root string `json:"root"`
		}
		e := json.Unmarshal(raw, &v)
		d.Root = v.Root
		return d, e
	}
	if keyID == nil {
		return d, errors.New("missing provider credentials")
	}
	key, e := r.Keys.Get(ctx, *keyID)
	if e != nil {
		return d, e
	}
	plain, e := cryptobox.Open(key, "filex/storage/v1/"+id, box)
	if e != nil {
		return d, e
	}
	d.S3 = &S3Config{}
	e = json.Unmarshal(plain, d.S3)
	return d, e
}

type Inventory interface {
	List(context.Context, string, int) ([]string, string, error)
}

func (l Local) List(ctx context.Context, cursor string, limit int) ([]string, string, error) {
	if limit < 1 || limit > 1000 {
		return nil, "", errors.New("invalid inventory limit")
	}
	root, e := l.root()
	if e != nil {
		return nil, "", e
	}
	defer root.Close()
	dir, e := root.Open(".")
	if e != nil {
		return nil, "", e
	}
	defer dir.Close()
	keys := []string{}
	count := 0
	for {
		if e = ctx.Err(); e != nil {
			return nil, "", e
		}
		entries, err := dir.ReadDir(256)
		for _, entry := range entries {
			count++
			if count > 1000000 {
				return nil, "", errors.New("inventory object limit exceeded")
			}
			if !entry.IsDir() && entry.Name() > cursor {
				keys = append(keys, entry.Name())
			}
		}
		sort.Strings(keys)
		if len(keys) > limit+1 {
			keys = keys[:limit+1]
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
	}
	next := ""
	if len(keys) > limit {
		keys = keys[:limit]
		next = keys[len(keys)-1]
	}
	return keys, next, nil
}
func (s S3) List(ctx context.Context, cursor string, limit int) ([]string, string, error) {
	if limit < 1 || limit > 1000 {
		return nil, "", errors.New("invalid inventory limit")
	}
	in := &s3.ListObjectsV2Input{Bucket: &s.Bucket, MaxKeys: aws.Int32(int32(limit))}
	if cursor != "" {
		in.ContinuationToken = &cursor
	}
	out, e := s.Client.ListObjectsV2(ctx, in)
	if e != nil {
		return nil, "", e
	}
	keys := []string{}
	for _, o := range out.Contents {
		if o.Key != nil {
			keys = append(keys, *o.Key)
		}
	}
	next := ""
	if aws.ToBool(out.IsTruncated) {
		next = aws.ToString(out.NextContinuationToken)
		if next == "" {
			return nil, "", errors.New("missing inventory continuation")
		}
	}
	return keys, next, nil
}
