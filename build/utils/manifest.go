package utils

import (
	// sha1 is not cryptographically secure
	// but we just want to generate a reproducible fast hash
	// to compare the contents of the files
	"crypto/sha1" // nolint:gosec
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func GenerateManifest() (string, error) {
	manifest := ""
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			hash, err := insecureHashFileContent(path)
			if err != nil {
				return err
			}
			manifest = manifest + hash + ":" + path + "\n"
		}
		return nil
	})

	return manifest, err
}

// insecureHashFileContent returns the SHA1 hash of the file content.
// It is not cryptographically secure. do not use for anything else
func insecureHashFileContent(path string) (string, error) {
	// Handle hashing big files.
	// Source: https://stackoverflow.com/q/60328216/1722542

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		err = f.Close()
	}()

	buf := make([]byte, 1024*1024)
	h := sha1.New() // nolint:gosec

	for {
		bytesRead, err := f.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return "", err
			}
			_, err = h.Write(buf[:bytesRead])
			if err != nil {
				return "", err
			}
			break
		}
		_, err = h.Write(buf[:bytesRead])
		if err != nil {
			return "", err
		}
	}

	fileHash := hex.EncodeToString(h.Sum(nil))
	return fileHash, nil
}
