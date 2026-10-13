package transfer

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrLocked = errors.New("resource locked")

type davGrantKey struct{}
type davGrant struct {
	Credential string
	Tokens     map[string]bool
}

// WithDAVGrant carries only tokens whose positive If condition matched a resource.
// Parent-locked mutation paths recheck the current durable lock on every write.
func WithDAVGrant(ctx context.Context, credential string, tokens map[string]bool) context.Context {
	return context.WithValue(ctx, davGrantKey{}, davGrant{credential, tokens})
}

type DAVLock struct {
	Token, Credential, Path, Key, OwnerXML string
	Recursive                              bool
	Expires                                time.Time
}

type DAVQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func DAVLocks(ctx context.Context, q DAVQuerier, id string) ([]DAVLock, error) {
	rows, e := q.Query(ctx, `SELECT l.token::text,l.credential_id::text,l.path,l.path_key,l.recursive,l.owner_xml,l.expires_at FROM dav_locks l JOIN api_tokens c ON c.id=l.credential_id JOIN users u ON u.id=c.user_id WHERE l.transfer_id=$1 AND l.expires_at>clock_timestamp() AND c.expires_at>clock_timestamp() AND c.revoked_at IS NULL AND c.auth_epoch=u.auth_epoch AND NOT u.disabled ORDER BY l.token`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []DAVLock{}
	for rows.Next() {
		var l DAVLock
		if e = rows.Scan(&l.Token, &l.Credential, &l.Path, &l.Key, &l.Recursive, &l.OwnerXML, &l.Expires); e != nil {
			return nil, e
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
func DAVCovers(root, key string, recursive bool) bool {
	return root == key || recursive && (root == "" || strings.HasPrefix(key, root+"/"))
}
func CheckDAVLocks(ctx context.Context, tx pgx.Tx, id, key string, subtree bool) error {
	locks, e := DAVLocks(ctx, tx, id)
	if e != nil {
		return e
	}
	grant, _ := ctx.Value(davGrantKey{}).(davGrant)
	for _, l := range locks {
		if DAVCovers(l.Key, key, l.Recursive) || subtree && DAVCovers(key, l.Key, true) {
			if grant.Credential != l.Credential || !grant.Tokens[l.Token] {
				return ErrLocked
			}
		}
	}
	return nil
}

// A Depth:0 collection lock protects its immediate membership, not child bytes.
func CheckDAVMembership(ctx context.Context, tx pgx.Tx, id, key string) error {
	parent := path.Dir(key)
	if parent == "." {
		parent = ""
	}
	return CheckDAVLocks(ctx, tx, id, parent, false)
}
