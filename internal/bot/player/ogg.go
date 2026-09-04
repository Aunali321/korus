package player

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

const (
	oggHeaderSize   = 27
	lacingContinued = 255
)

var errNotOgg = errors.New("player: not an ogg stream")

// oggReader pulls Opus packets out of the Ogg container FFmpeg writes to stdout.
// Packets span one or more lacing segments and may continue across pages.
type oggReader struct {
	src     *bufio.Reader
	packets [][]byte
	partial []byte
}

func newOggReader(r io.Reader) *oggReader {
	return &oggReader{src: bufio.NewReaderSize(r, 64<<10)}
}

// next returns the next Opus packet, skipping the two Ogg Opus header packets.
func (o *oggReader) next() ([]byte, error) {
	for {
		if len(o.packets) > 0 {
			packet := o.packets[0]
			o.packets = o.packets[1:]
			if isOpusHeader(packet) {
				continue
			}
			return packet, nil
		}
		if err := o.readPage(); err != nil {
			return nil, err
		}
	}
}

func (o *oggReader) readPage() error {
	var header [oggHeaderSize]byte
	if _, err := io.ReadFull(o.src, header[:]); err != nil {
		return err
	}
	if string(header[0:4]) != "OggS" {
		return errNotOgg
	}

	table := make([]byte, header[26])
	if _, err := io.ReadFull(o.src, table); err != nil {
		return fmt.Errorf("read segment table: %w", err)
	}
	size := 0
	for _, segment := range table {
		size += int(segment)
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(o.src, body); err != nil {
		return fmt.Errorf("read page body: %w", err)
	}

	offset := 0
	for _, segment := range table {
		o.partial = append(o.partial, body[offset:offset+int(segment)]...)
		offset += int(segment)
		if segment < lacingContinued {
			o.packets = append(o.packets, o.partial)
			o.partial = nil
		}
	}
	return nil
}

func isOpusHeader(packet []byte) bool {
	if len(packet) < 8 {
		return false
	}
	prefix := string(packet[:8])
	return prefix == "OpusHead" || prefix == "OpusTags"
}
