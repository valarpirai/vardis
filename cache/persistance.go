package cache

import (
	"bufio"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

// AOF Implementation
// Log Create, Update, Delete actions to File
// Write entire command to the file
// Read and load the file on startup
type Persistance struct {
	flushInterval time.Duration
	AofFile       *os.File
}

// Initialize Persistant store
func NewStorage() *Persistance {
	persist := new(Persistance)
	persist.flushInterval = 3
	file_name := "appendonly.aof"
	f, err := os.OpenFile(file_name, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Error(err)
	}
	persist.AofFile = f
	go persist.flush()
	return persist
}

func (p *Persistance) flush() {
	// File sync
	time.Sleep(p.flushInterval * time.Second)
	p.AofFile.Sync()
	p.flush()
}

func (p *Persistance) WriteCommand(cmd string) {
	p.AofFile.WriteString(cmd)
}

func (p *Persistance) Reader() *bufio.Reader {
	return bufio.NewReader(p.AofFile)
}
