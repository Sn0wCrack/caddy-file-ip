package caddy_file_ip

import (
	"bufio"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/fsnotify/fsnotify"
)

func init() {
	caddy.RegisterModule(FileIPSource{})
}

type FileIPSource struct {
	Files    []string       `json:"files,omitempty"`
	Watch    bool           `json:"watch,omitempty"`
	Interval caddy.Duration `json:"interval,omitempty"`

	ranges  []netip.Prefix
	ctx     caddy.Context
	lock    *sync.RWMutex
	watcher *fsnotify.Watcher
	done    chan struct{}
}

func (FileIPSource) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.ip_sources.file",
		New: func() caddy.Module { return new(FileIPSource) },
	}
}

func (s *FileIPSource) Provision(ctx caddy.Context) error {
	s.ctx = ctx
	s.lock = new(sync.RWMutex)
	s.done = make(chan struct{})

	if err := s.loadRanges(); err != nil {
		return err
	}

	if s.Watch {
		if err := s.startWatcher(); err != nil {
			return err
		}
	} else if s.Interval > 0 {
		go s.startTimer()
	}

	return nil
}

func (s *FileIPSource) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	s.watcher = watcher

	for _, file := range s.Files {
		if err := watcher.Add(file); err != nil {
			watcher.Close()
			return err
		}
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					s.lock.Lock()
					s.loadRanges()
					s.lock.Unlock()
				}
			case <-s.done:
				watcher.Close()
				return
			}
		}
	}()

	return nil
}

func (s *FileIPSource) startTimer() {
	if s.Interval == 0 {
		s.Interval = caddy.Duration(time.Hour)
	}

	ticker := time.NewTicker(time.Duration(s.Interval))

	for {
		select {
		case <-ticker.C:
			s.lock.Lock()
			s.loadRanges()
			s.lock.Unlock()
		case <-s.done:
			ticker.Stop()
			return
		}
	}
}

func (s *FileIPSource) loadRanges() error {
	var allPrefixes []netip.Prefix

	for _, filePath := range s.Files {
		prefixes, err := s.readFile(filePath)
		if err != nil {
			continue
		}
		allPrefixes = append(allPrefixes, prefixes...)
	}

	s.ranges = allPrefixes
	return nil
}

func (s *FileIPSource) readFile(filePath string) ([]netip.Prefix, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var prefixes []netip.Prefix
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		prefix, err := caddyhttp.CIDRExpressionToPrefix(line)
		if err != nil {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes, scanner.Err()
}

func (s *FileIPSource) GetIPRanges(_ *http.Request) []netip.Prefix {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ranges
}

func (s *FileIPSource) Stop() error {
	close(s.done)
	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}

var (
	_ caddy.Module            = (*FileIPSource)(nil)
	_ caddy.Provisioner       = (*FileIPSource)(nil)
	_ caddyhttp.IPRangeSource = (*FileIPSource)(nil)
	_ caddyfile.Unmarshaler   = (*FileIPSource)(nil)
)
