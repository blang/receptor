package filelog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/blang/receptor/event"
	"github.com/blang/receptor/handler"
	"os"
	"time"
)

type FileLogReactor struct {
}

type ServiceConfig struct {
	Filename   string `json:"filename"`
	Unbuffered bool   `json:"unbuffered"`
}

func (r *FileLogReactor) Setup(_ json.RawMessage) error {
	// No global config needed
	return nil
}
func (r *FileLogReactor) Accept(cfgData json.RawMessage) (handler.Handler, error) {
	var cfg ServiceConfig
	err := json.Unmarshal(cfgData, &cfg)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(cfg.Filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	bufW := bufio.NewWriter(f)

	return handler.HandlerFunc(func(eventCh chan event.Event, doneCh chan struct{}) {
		for e := range eventCh {
			for _, node := range e.Nodes() {
				fmt.Fprintf(bufW, "%s: %s (%s) %s:%d\n", time.Now(), node.Name(), node.Type(), node.Host(), node.Port())
			}
			if cfg.Unbuffered {
				bufW.Flush()
			}
		}
		bufW.Flush()
		f.Close()
		close(doneCh)
	}), nil
}