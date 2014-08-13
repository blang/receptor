package plugin

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFindExecutableNotExecutable(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()
	l := Lookup{
		pluginPath: filepath.Dir(tmpFile.Name()),
	}
	_, err = l.findExecutable(filepath.Base(tmpFile.Name()))
	if err == nil {
		t.Error("Expected error, file is not executable")
	}
}

func TestFindExecutableDirectory(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpDir)
	l := Lookup{
		pluginPath: filepath.Dir(tmpDir),
	}
	_, err = l.findExecutable(filepath.Base(tmpDir))
	if err == nil {
		t.Error("Expected error, file is not executable")
	}
}

func TestFindExecutable(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()
	l := Lookup{
		pluginPath: filepath.Dir(tmpFile.Name()),
	}

	os.Chmod(tmpFile.Name(), 0700)

	_, err = l.findExecutable(filepath.Base(tmpFile.Name()))
	if err != nil {
		t.Error("No error expected, file is now executable")
	}
}
