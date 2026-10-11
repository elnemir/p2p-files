package files

import (
	"context"
	"io"
	"strings"
)

// Scanner must return an error for infection, timeout or an indeterminate result.
// The callback streams plaintext without persisting it to a temporary file.
type Scanner interface {
	Scan(context.Context, string, func(io.Writer) error) error
}

func permitted(mode string, types []string, media string) bool {
	media = strings.Split(media, ";")[0]
	found := false
	for _, v := range types {
		if v == media || strings.HasSuffix(v, "/*") && strings.HasPrefix(media, strings.TrimSuffix(v, "*")) {
			found = true
		}
	}
	return mode == "" || mode == "disabled" || mode == "allow" && found || mode == "deny" && !found
}
