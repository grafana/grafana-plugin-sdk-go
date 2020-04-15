package utils

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFile(t *testing.T) {
	src, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.RemoveAll(src.Name())
	err = ioutil.WriteFile(src.Name(), []byte("Contents"), 0600)
	require.NoError(t, err)

	dst, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dst.Name())

	err = CopyFile(src.Name(), dst.Name())
	require.NoError(t, err)
}

// Test case where destination directory doesn't exist.
func TestCopyFile_NonExistentDestDir(t *testing.T) {
	src, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.RemoveAll(src.Name())

	err = CopyFile(src.Name(), "non-existent/dest")
	require.EqualError(t, err, "destination directory doesn't exist: \"non-existent\"")
}

func TestCopyRecursive_NonExistentDest(t *testing.T) {
	src, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(src)

	err = os.MkdirAll(path.Join(src, "data"), 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile(path.Join(src, "data", "file.txt"), []byte("Test"), 0644)
	require.NoError(t, err)

	dstParent, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dstParent)

	dst := path.Join(dstParent, "dest")

	err = CopyRecursive(src, dst)
	require.NoError(t, err)

	compareDirs(t, src, dst)
}

func TestCopyRecursive_ExistentDest(t *testing.T) {
	src, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(src)

	err = os.MkdirAll(path.Join(src, "data"), 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile(path.Join(src, "data", "file.txt"), []byte("Test"), 0644)
	require.NoError(t, err)

	dst, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dst)

	err = CopyRecursive(src, dst)
	require.NoError(t, err)

	compareDirs(t, src, dst)
}

func compareDirs(t *testing.T, src, dst string) {
	sfi, err := os.Stat(src)
	require.NoError(t, err)
	dfi, err := os.Stat(dst)
	require.NoError(t, err)

	require.Equal(t, sfi.Mode(), dfi.Mode())

	err = filepath.Walk(src, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(srcPath, src)
		dstPath := path.Join(dst, relPath)
		sfi, err := os.Stat(srcPath)
		require.NoError(t, err)

		dfi, err := os.Stat(dstPath)
		require.NoError(t, err)
		require.Equal(t, sfi.Mode(), dfi.Mode())

		if sfi.IsDir() {
			return nil
		}

		srcData, err := ioutil.ReadFile(srcPath)
		require.NoError(t, err)
		dstData, err := ioutil.ReadFile(dstPath)
		require.NoError(t, err)

		require.Equal(t, srcData, dstData)

		return nil
	})
	require.NoError(t, err)
}
