package plugin

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
)

// Socket creates a new socket and returns the socket path.
func newSocket() (string, error) {
	tmpFile, err := ioutil.TempFile("", "receptor-plugin")
	if err != nil {
		return "", err
	}

	socketPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(tmpFile.Name()); err != nil {
		return "", err
	}

	return socketPath, nil
}

func extrSocketInfo(args []string) (string, string, error) {
	if len(args) != 3 {
		return "", "", errors.New("Wrong arguments")
	}
	return args[1], args[2], nil
}

func setLogFlags() {
	log.SetFlags(0)
}
