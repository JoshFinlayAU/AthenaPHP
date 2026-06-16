// Package encoder turns PHP source files into Athena containers and can encode
// an entire project tree in place or into an output directory.
package encoder

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"athena/internal/crypto"
	"athena/internal/format"
	"athena/internal/walker"
)

// stub is the ASCII PHP prologue shown only when the extension is absent. It
// contains no NUL/0x01 bytes, so the binary MAGIC never collides with it.
const stub = `<?php
/* Athena-encoded file. Requires the 'athena' PHP extension to run. */
if (!extension_loaded('athena')) {
    throw new \RuntimeException('Athena: the athena extension is not loaded; cannot run this encoded file.');
}
__halt_compiler();
`

// EncodeBytes encrypts one PHP source buffer into a complete encoded file
// (ASCII stub followed by the binary container).
func EncodeBytes(src, key []byte) ([]byte, error) {
	hdr := format.BuildHeader(format.Header{
		Version: format.Version,
		Flags:   0,
		KeyID:   format.KeyID(key),
		OrigLen: uint32(len(src)),
	})
	nonce, ct, err := crypto.Seal(key, src, hdr)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString(stub)
	buf.Write(hdr)   // includes MAGIC at offset 0
	buf.Write(nonce) // 24 bytes
	buf.Write(ct)    // ciphertext + 16-byte tag
	return buf.Bytes(), nil
}

// Already reports whether b is already an Athena-encoded file.
func Already(b []byte) bool {
	return bytes.Contains(b, format.Magic)
}

// Stats summarizes an encode run.
type Stats struct {
	Encoded int
	Skipped int
	Bytes   int64
}

// EncodeProject encodes a project. If dst is empty it rewrites matching .php
// files in place (non-PHP and vendor/ files are left untouched). Otherwise it
// mirrors the entire tree into dst — encoding eligible .php files and copying
// everything else (assets, vendor/, etc.) verbatim, yielding a deployable copy.
func EncodeProject(src, dst string, key []byte, opt walker.Options, log func(string)) (Stats, error) {
	if dst == "" {
		return encodeInPlace(src, key, opt, log)
	}
	return encodeToDir(src, dst, key, opt, log)
}

func encodeInPlace(src string, key []byte, opt walker.Options, log func(string)) (Stats, error) {
	var st Stats
	files, err := walker.Walk(src, opt)
	if err != nil {
		return st, err
	}
	for _, rel := range files {
		inPath := filepath.Join(src, rel)
		data, err := os.ReadFile(inPath)
		if err != nil {
			return st, fmt.Errorf("read %s: %w", rel, err)
		}
		if Already(data) {
			st.Skipped++
			logf(log, "skip  %s (already encoded)", rel)
			continue
		}
		enc, err := EncodeBytes(data, key)
		if err != nil {
			return st, fmt.Errorf("encode %s: %w", rel, err)
		}
		if err := writeFile(inPath, enc, inPath); err != nil {
			return st, err
		}
		st.Encoded++
		st.Bytes += int64(len(enc))
		logf(log, "enc   %s (%d -> %d bytes)", rel, len(data), len(enc))
	}
	return st, nil
}

func encodeToDir(src, dst string, key []byte, opt walker.Options, log func(string)) (Stats, error) {
	var st Stats
	files, err := walker.WalkAll(src)
	if err != nil {
		return st, err
	}
	for _, rel := range files {
		inPath := filepath.Join(src, rel)
		outPath := filepath.Join(dst, rel)
		data, err := os.ReadFile(inPath)
		if err != nil {
			return st, fmt.Errorf("read %s: %w", rel, err)
		}
		if opt.ShouldEncode(rel) && !Already(data) {
			enc, err := EncodeBytes(data, key)
			if err != nil {
				return st, fmt.Errorf("encode %s: %w", rel, err)
			}
			if err := writeFile(outPath, enc, inPath); err != nil {
				return st, err
			}
			st.Encoded++
			st.Bytes += int64(len(enc))
			logf(log, "enc   %s (%d -> %d bytes)", rel, len(data), len(enc))
			continue
		}
		if err := writeFile(outPath, data, inPath); err != nil {
			return st, err
		}
		st.Skipped++
		logf(log, "copy  %s", rel)
	}
	return st, nil
}

func logf(log func(string), format string, a ...any) {
	if log != nil {
		log(fmt.Sprintf(format, a...))
	}
}

// writeFile writes data to path, creating parents and preserving the source
// file's mode.
func writeFile(path string, data []byte, modeFrom string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(modeFrom); err == nil {
		mode = fi.Mode().Perm()
	}
	return os.WriteFile(path, data, mode)
}
