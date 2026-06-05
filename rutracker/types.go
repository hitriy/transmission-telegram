package rutracker

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
)

const (
	Host      = "https://rutracker.org"
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type Torrent struct {
	ID         string
	Title      string
	Author     string
	Category   string
	Size       int64
	Seeds      int
	Leeches    int
	Downloads  int
	State      string
	Registered time.Time
}

func (t Torrent) FormattedSize() string {
	return humanize.Bytes(uint64(t.Size))
}

func (t Torrent) URL() string {
	return fmt.Sprintf("%s/forum/viewtopic.php?t=%s", Host, t.ID)
}

type Thread struct {
	ID          string
	Title       string
	Magnet      string
	Description string
}
