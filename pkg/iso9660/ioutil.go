// Copyright © 2018 NVIDIA Corporation

package iso9660

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

func putBothUint16(w io.Writer, v uint16) error {
	if err := binary.Write(w, binary.LittleEndian, v); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return err
	}

	return nil
}

func putBothUint32(w io.Writer, v uint32) error {
	if err := binary.Write(w, binary.LittleEndian, v); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return err
	}

	return nil
}

func pad(w io.Writer, count int) error {
	for i := 0; i < count; i++ {
		if _, err := w.Write([]byte{0}); err != nil {
			return err
		}
	}
	return nil
}

func writeByte(w io.Writer, b byte) (err error) {
	_, err = w.Write([]byte{b})
	return
}

// Calculates the number of sectors needed to hold bytes. Zero bytes result in one sector.
func bytesToSectors(bytes uint32) uint32 {
	sectors := bytes / LogicalBlockSize
	if (bytes%LogicalBlockSize) != 0 || sectors == 0 {
		sectors++
	}
	return sectors
}

// Calculates the number of bytes occuppied by sectors
func sectorsToBytes(sectors uint32) uint64 {
	return uint64(sectors) * LogicalBlockSize
}

func readByte(r io.Reader) (b byte, err error) {
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return
	}

	b = buf[0]
	return
}

func readExpectedByte(r io.Reader, expected byte, desc string) error {
	var actual [1]byte
	_, err := io.ReadFull(r, actual[:])
	if err != nil {
		return err
	}

	if actual[0] != expected {
		return fmt.Errorf("%s: expected=%d, got=%d", desc, expected, actual[0])
	}

	return nil
}

func readExpectedString(r io.Reader, expected string, desc string) error {
	actual := make([]byte, len([]byte(expected)))
	_, err := io.ReadFull(r, actual[:])
	if err != nil {
		return err
	}

	if string(actual) != expected {
		return fmt.Errorf("%s: expected=%q, got=%q", desc, expected, string(actual))
	}

	return nil
}

func readStrA(r io.Reader, size int) (string, error) {
	buf := make([]byte, size)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}

	s := strings.ToUpper(strings.TrimRight(string(buf), " "))
	for _, r := range s {
		if _, ok := StrARunes[r]; !ok {
			return "", fmt.Errorf("StrA field contains invalid rune: %q", r)
		}
	}

	return s, nil
}

func readStrD(r io.Reader, size int) (string, error) {
	buf := make([]byte, size)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}

	s := strings.ToUpper(strings.TrimRight(string(buf), " "))
	for _, r := range s {
		if _, ok := StrDRunes[r]; !ok {
			return "", fmt.Errorf("StrD field contains invalid rune: %q", r)
		}
	}

	return s, nil
}

func getBothUint16(r io.Reader) (uint16, error) {
	var a uint16
	if err := binary.Read(r, binary.LittleEndian, &a); err != nil {
		return 0, err
	}

	var b uint16
	if err := binary.Read(r, binary.BigEndian, &b); err != nil {
		return 0, err
	}

	if a != b {
		return 0, fmt.Errorf("getBothUint16 mismatch: le=%d, be=%d", a, b)
	}

	return a, nil
}

func getBothUint32(r io.Reader) (uint32, error) {
	var a uint32
	if err := binary.Read(r, binary.LittleEndian, &a); err != nil {
		return 0, err
	}

	var b uint32
	if err := binary.Read(r, binary.BigEndian, &b); err != nil {
		return 0, err
	}

	if a != b {
		return 0, fmt.Errorf("getBothUint32 mismatch: le=%d, be=%d", a, b)
	}

	return a, nil
}

func unpad(r io.Reader, count int) error {
	_, err := io.CopyN(ioutil.Discard, r, int64(count))
	if err != nil {
		return err
	}

	return nil
}

type CountingReader struct {
	r io.Reader
	n uint64
}

func newCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{
		r: r,
	}
}

func (cr *CountingReader) Read(buf []byte) (int, error) {
	n, err := cr.r.Read(buf)
	cr.n += uint64(n)
	return n, err
}

func (cr *CountingReader) Consumed() uint64 {
	return cr.n
}

type CountingWriter struct {
	w io.Writer
	n int64
}

func newCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{
		w: w,
	}
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.n += int64(n)
	return
}

func (cw *CountingWriter) Written() int64 {
	return cw.n
}
