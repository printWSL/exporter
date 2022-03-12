package main

import (
	"os"

	"github.com/go-kit/log"
)

func main() {

	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	logger.Log("question", "what is the meaning of life?", "answer", 42)

}
