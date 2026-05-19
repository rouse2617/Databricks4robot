package remux

import "errors"

// foxglove.CompressedVideo proto schema (relevant fields only):
//
//   message CompressedVideo {
//     google.protobuf.Timestamp timestamp = 1;  // not used here
//     string  frame_id  = 2;
//     bytes   data      = 3;
//     string  format    = 4;
//   }
//
// We hand-decode rather than depending on a generated .pb.go to keep this
// service free of a protoc / protobuf-go build step. The wire format is
// well-defined and small enough that ~30 LOC suffices.
//
// All fields are wire type 2 (LEN). Unknown / unrecognized tags are skipped.

// DecodeFoxgloveCompressedVideo extracts the data and format fields from a
// serialized foxglove.CompressedVideo message.
func DecodeFoxgloveCompressedVideo(buf []byte) (data []byte, format string, err error) {
	for len(buf) > 0 {
		key, n := readVarint(buf)
		if n <= 0 {
			return nil, "", errors.New("compressed_video: bad tag varint")
		}
		buf = buf[n:]
		fieldNum := int(key >> 3)
		wireType := int(key & 0x7)

		switch wireType {
		case 0: // VARINT
			_, n = readVarint(buf)
			if n <= 0 {
				return nil, "", errors.New("compressed_video: bad varint payload")
			}
			buf = buf[n:]
		case 1: // I64
			if len(buf) < 8 {
				return nil, "", errors.New("compressed_video: short i64")
			}
			buf = buf[8:]
		case 2: // LEN
			ln, n := readVarint(buf)
			if n <= 0 {
				return nil, "", errors.New("compressed_video: bad length varint")
			}
			buf = buf[n:]
			if uint64(len(buf)) < ln {
				return nil, "", errors.New("compressed_video: short LEN payload")
			}
			payload := buf[:ln]
			buf = buf[ln:]
			switch fieldNum {
			case 3: // data
				data = payload
			case 4: // format
				format = string(payload)
			}
		case 5: // I32
			if len(buf) < 4 {
				return nil, "", errors.New("compressed_video: short i32")
			}
			buf = buf[4:]
		default:
			return nil, "", errors.New("compressed_video: unsupported wire type")
		}
	}
	return data, format, nil
}

// readVarint decodes a protobuf base-128 varint. Returns (value, bytes_consumed).
// Returns n<=0 on failure.
func readVarint(buf []byte) (uint64, int) {
	var v uint64
	var shift uint
	for i, b := range buf {
		if i >= 10 {
			return 0, -1
		}
		v |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return v, i + 1
		}
		shift += 7
	}
	return 0, -1
}
