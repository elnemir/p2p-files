package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
)

type davCondition struct {
	Value     string
	Not, ETag bool
}
type davIfList struct {
	Tag        string
	Conditions []davCondition
}

// The header grammar is bounded independently of the XML and namespace budgets.
func davParseIf(raw string) ([]davIfList, error) {
	if len(raw) > 8192 {
		return nil, davError(400)
	}
	rest := strings.TrimSpace(raw)
	out := []davIfList{}
	tag := ""
	tagged := strings.HasPrefix(rest, "<")
	take := func(open, close byte) (string, bool) {
		if len(rest) == 0 || rest[0] != open {
			return "", false
		}
		i := strings.IndexByte(rest[1:], close)
		if i < 0 {
			return "", false
		}
		value := rest[1 : i+1]
		rest = strings.TrimSpace(rest[i+2:])
		return value, value != ""
	}
	for rest != "" {
		if strings.HasPrefix(rest, "<") {
			if !tagged {
				return nil, davError(400)
			}
			var ok bool
			tag, ok = take('<', '>')
			if !ok {
				return nil, davError(400)
			}
		}
		if !strings.HasPrefix(rest, "(") {
			return nil, davError(400)
		}
		rest = strings.TrimSpace(rest[1:])
		list := davIfList{Tag: tag}
		for !strings.HasPrefix(rest, ")") {
			c := davCondition{}
			if strings.HasPrefix(rest, "Not ") || strings.HasPrefix(rest, "Not\t") {
				c.Not = true
				rest = strings.TrimSpace(rest[3:])
			}
			var ok bool
			if strings.HasPrefix(rest, "[") {
				c.ETag = true
				c.Value, ok = take('[', ']')
			} else {
				c.Value, ok = take('<', '>')
			}
			if !ok {
				return nil, davError(400)
			}
			list.Conditions = append(list.Conditions, c)
		}
		if len(list.Conditions) == 0 {
			return nil, davError(400)
		}
		rest = strings.TrimSpace(rest[1:])
		out = append(out, list)
	}
	return out, nil
}

func davETagMatches(header, etag string, exists, weak bool) bool {
	for _, v := range strings.Split(header, ",") {
		v = strings.TrimSpace(v)
		if v == "*" && exists {
			return true
		}
		if !exists {
			continue
		}
		if weak {
			v = strings.TrimPrefix(v, "W/")
			etag = strings.TrimPrefix(etag, "W/")
		}
		if v == etag && !strings.HasPrefix(v, "W/") {
			return true
		}
	}
	return false
}

func (d davAccess) conditions(ctx context.Context, q davQuerier, r *http.Request, name string) error {
	node, e := d.lookup(ctx, q, name)
	exists := e == nil
	if e != nil && !errors.Is(e, davError(404)) {
		return e
	}
	if v := r.Header.Get("If-Match"); v != "" && !davETagMatches(v, node.etag(), exists, false) {
		return davError(412)
	}
	if v := r.Header.Get("If-None-Match"); v != "" && davETagMatches(v, node.etag(), exists, true) {
		if r.Method == "GET" || r.Method == "HEAD" {
			return davError(304)
		}
		return davError(412)
	}
	if r.Header.Get("If-Match") == "" {
		if date, e := http.ParseTime(r.Header.Get("If-Unmodified-Since")); e == nil && exists && node.Modified.Unix() > date.Unix() {
			return davError(412)
		}
	}
	if r.Header.Get("If-None-Match") == "" && (r.Method == "GET" || r.Method == "HEAD") {
		if date, e := http.ParseTime(r.Header.Get("If-Modified-Since")); e == nil && exists && node.Modified.Unix() <= date.Unix() {
			return davError(304)
		}
	}
	clear(d.tokens)
	lists, e := davParseIf(r.Header.Get("If"))
	if e != nil {
		return e
	}
	if len(lists) == 0 {
		return nil
	}
	locks, e := transfer.DAVLocks(ctx, q, d.transfer)
	if e != nil {
		return e
	}
	any := false
	for _, list := range lists {
		resource := name
		if list.Tag != "" {
			u, e := url.Parse(list.Tag)
			if e != nil {
				return davError(400)
			}
			origin, _ := url.Parse(d.settings.PublicOrigin)
			if u.IsAbs() && (u.Scheme != origin.Scheme || !strings.EqualFold(u.Host, origin.Host)) {
				return davError(412)
			}
			id, p, e := davPath(u, d.settings.MaxPathBytes)
			if e != nil || id != d.transfer {
				return davError(412)
			}
			resource = p
		}
		key := ""
		if resource != "" {
			key, e = domain.PathKey(resource, d.settings.MaxPathBytes)
			if e != nil {
				return e
			}
		}
		n, err := d.lookup(ctx, q, resource)
		if err != nil && !errors.Is(err, davError(404)) {
			return err
		}
		match := true
		tokens := []string{}
		for _, c := range list.Conditions {
			value := false
			if c.ETag {
				value = davETagMatches(c.Value, n.etag(), err == nil, true)
			} else {
				for _, l := range locks {
					if c.Value == "urn:uuid:"+l.Token && transfer.DAVCovers(l.Key, key, l.Recursive) {
						value = true
						if !c.Not && l.Credential == d.credential {
							tokens = append(tokens, l.Token)
						}
					}
				}
			}
			if c.Not {
				value = !value
			}
			match = match && value
		}
		if match {
			any = true
			for _, t := range tokens {
				d.tokens[t] = true
			}
		}
	}
	if !any {
		return davError(412)
	}
	return nil
}
