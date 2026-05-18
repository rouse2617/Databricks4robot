package gcsrs

import "strings"

// ParseGSURI splits gs://bucket/object/key into bucket and object name.
func ParseGSURI(uri string) (bucket, object string, ok bool) {
	const prefix = "gs://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(uri, prefix)
	i := strings.IndexByte(rest, '/')
	if i <= 0 || i >= len(rest)-1 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}
