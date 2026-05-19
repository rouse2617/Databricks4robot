package main

import (
	"fmt"
	"io"
)

// countingReadSeeker wraps io.ReadSeeker and records how the consumer (e.g. mcap.Reader.Info)
// uses Read vs Seek — same pattern for local disk or GCS-backed seeker.
type countingReadSeeker struct {
	inner io.ReadSeeker

	ReadCalls int64
	SeekCalls int64
	BytesRead int64

	SeekBack int64 // new pos < old pos
	SeekFwd  int64 // new pos > old pos
	SeekSame int64
	MinRead  int64 // min n for calls where n > 0
	MaxRead  int64

	pos int64 // logical position we track (must match inner after each op)
}

func newCountingReadSeeker(inner io.ReadSeeker) *countingReadSeeker {
	return &countingReadSeeker{
		inner:   inner,
		MinRead: 1 << 62,
	}
}

func (c *countingReadSeeker) Read(p []byte) (int, error) {
	n, err := c.inner.Read(p)
	c.ReadCalls++
	c.BytesRead += int64(n)
	if n > 0 {
		c.pos += int64(n)
		if int64(n) < c.MinRead {
			c.MinRead = int64(n)
		}
		if int64(n) > c.MaxRead {
			c.MaxRead = int64(n)
		}
	}
	return n, err
}

func (c *countingReadSeeker) Seek(offset int64, whence int) (int64, error) {
	old := c.pos
	pos, err := c.inner.Seek(offset, whence)
	if err != nil {
		return pos, err
	}
	c.pos = pos
	c.SeekCalls++
	switch {
	case pos < old:
		c.SeekBack++
	case pos > old:
		c.SeekFwd++
	default:
		c.SeekSame++
	}
	return pos, err
}

func (c *countingReadSeeker) Summary() string {
	minStr := "—"
	if c.ReadCalls > 0 && c.MinRead < 1<<62 {
		minStr = fmt.Sprintf("%d", c.MinRead)
	}
	avg := float64(0)
	if c.ReadCalls > 0 {
		avg = float64(c.BytesRead) / float64(c.ReadCalls)
	}
	return fmt.Sprintf(
		"ReadSeeker stats: read_calls=%d seek_calls=%d bytes_via_read=%d (avg_read=%.1f B min=%s max=%d) | seek: back=%d fwd=%d same=%d",
		c.ReadCalls, c.SeekCalls, c.BytesRead, avg, minStr, c.MaxRead,
		c.SeekBack, c.SeekFwd, c.SeekSame,
	)
}
