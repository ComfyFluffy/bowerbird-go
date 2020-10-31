package downloader

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/helper"
)

type taskState int

// States of tasks
const (
	Pending taskState = iota
	Running
	Finished
	Paused
	Canceled
	Failed
	Skipped
)

const (
	defaultRetryMax     = 30
	defaultRetryWaitMin = 1 * time.Second
	defaultRetryWaitMax = 60 * time.Second
)

// Backoff calculates time to wait for the next retry.
type Backoff func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration

// Task stores a http download task, witch will be processed by Downloader
type Task struct {
	bytesNow int64 // bytesNow saves the downloaded bytes in this second.

	BytesLastSec int64
	Err          error
	Status       taskState
	Request      *http.Request
	LocalPath    string
	Overwrite    bool

	AfterFinished func(*Task)
}

func (t *Task) copy(dst io.Writer, src io.Reader, bytesChan chan int64) (written int64, err error) {
	// set the t.bytesNow to t.BytesLastSec and clear it every second
	bytesTicker := time.NewTicker(1 * time.Second)
	defer func() {
		bytesTicker.Stop()
		t.BytesLastSec = 0
		t.bytesNow = 0
	}()
	buf := make([]byte, 32*1024)

	for {
		nr, er := src.Read(buf)

		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				n := int64(nw)
				written += n

				select {
				case <-bytesTicker.C:
					t.BytesLastSec = t.bytesNow
					t.bytesNow = 0
				default:
					t.bytesNow += n
					// push n to global speed calculating goroutine
					bytesChan <- n
				}
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// Downloader processes the added tasks and save them to disk.
type Downloader struct {
	runningWorkers    int
	stopAll           chan struct{}
	globleBytesTicker *time.Ticker
	bytesChan         chan int64
	bytesNow,
	// bytes downloaded in the last second
	BytesLastSec int64
	Logger *log.Logger

	in    chan *Task
	Tasks []*Task

	wg           sync.WaitGroup
	once         sync.Once
	Client       *http.Client
	TriesMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	Backoff      Backoff
	MaxWorkers   int
}

func (d *Downloader) worker() {
	for {
		select {
		case <-d.stopAll:
			return
		case t := <-d.in:
			d.Download(t)
			d.wg.Done()
			// d.out <- t
		}
	}
}

// Start runs background goroutines of the downloader.
func (d *Downloader) Start() {
	d.once.Do(func() {
		d.Logger.Debug(fmt.Sprintf("starting downloader"))

		d.globleBytesTicker = time.NewTicker(1 * time.Second)

		go func() {
			for {
				select {
				case b := <-d.bytesChan:
					// calculate the global donwload speed
					d.bytesNow += b
				case <-d.globleBytesTicker.C:
					d.BytesLastSec = d.bytesNow
					d.bytesNow = 0
				case <-d.stopAll:
					d.bytesNow = 0
					d.BytesLastSec = 0
					d.once = sync.Once{}
					return
				}
			}
		}()

		for i := 0; i < d.MaxWorkers; i++ {
			go d.worker()
		}
		d.runningWorkers = d.MaxWorkers
	})
}

// Stop terminates background goroutines and tickers.
func (d *Downloader) Stop() {
	d.Logger.Debug("Stopping downloader")
	close(d.stopAll)
	d.globleBytesTicker.Stop()
}

// Add pushes the task to the downloader queue.
func (d *Downloader) Add(task *Task) {
	d.Tasks = append(d.Tasks, task)
	d.wg.Add(1)
	go func() {
		d.in <- task
	}()
}

// Wait blocks until all tasks are done.
func (d *Downloader) Wait() {
	d.wg.Wait()
}

// NewWithDefaultClient calls NewWithCliet with an empty *http.Client.
func NewWithDefaultClient() *Downloader {
	return NewWithCliet(&http.Client{Transport: &http.Transport{}})
}

// NewWithCliet builds a new Downloader with default value
// with the given *http.Client
func NewWithCliet(c *http.Client) *Downloader {
	return &Downloader{
		TriesMax:     defaultRetryMax,
		RetryWaitMax: defaultRetryWaitMax,
		RetryWaitMin: defaultRetryWaitMin,
		Backoff:      helper.DefaultBackoff,
		Client:       c,
		Logger:       log.G,
		in:           make(chan *Task, 65535),
		bytesChan:    make(chan int64),
		stopAll:      make(chan struct{}),
		Tasks:        []*Task{},
		MaxWorkers:   2,
	}
}

// Download starts downloading the given task.
func (d *Downloader) Download(t *Task) {
	if !t.Overwrite {
		if _, err := os.Stat(t.LocalPath); !os.IsNotExist(err) {
			t.Status = Finished
			if t.AfterFinished != nil {
				t.AfterFinished(t)
			}
			return
		}
	}

	onErr := func(message string, err error) {
		d.Logger.Error(fmt.Sprintf("task failed: download %s to %s: %s: %s", t.Request.URL, t.LocalPath, message, err))
		t.Status = Failed
		t.Err = err
	}

	t.Status = Running

	ctx := t.Request.Context()
	req := t.Request.Clone(ctx)
	tries := 0
	bytes := int64(0)
	part := t.LocalPath + ".part"

	log.G.Debug("starting task", req.URL, t.LocalPath, req.Header)

	os.MkdirAll(filepath.Dir(t.LocalPath), 0755)
	f, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		onErr("open file", err)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		d.Logger.Error("stat file", part, err)
	} else {
		// get the size of downloaded part
		bytes = fi.Size()
	}

	for {
		if bytes > 0 {
			// skip the downloaded part
			req.Header["Range"] = []string{fmt.Sprintf("bytes=%d-", bytes)}
			d.Logger.Debug("trying again with header:", req.Header)
		}

		tries++
		if tries > d.TriesMax {
			onErr("max tries", err)
			return
		}
		if tries > 1 {
			select {
			case <-ctx.Done():
				d.Logger.Debug("task canceled by context:", ctx.Err())
				t.Status = Canceled
				return
			case <-time.After(d.Backoff(d.RetryWaitMin, d.RetryWaitMax, tries, nil)):
			}
		}

		resp, err := d.Client.Do(req)
		if err != nil {
			d.Logger.Debug("response error:", err, req.URL)
			continue
		}
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			r, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				err = fmt.Errorf("http code %d with reading error: %w", resp.StatusCode, err)
			} else {
				err = fmt.Errorf("http code %d with message: %s", resp.StatusCode, r)
			}

			onErr("not http code 2xx", err)
			return
		}

		fn := filepath.Base(t.LocalPath)
		if resp.ContentLength == -1 {
			d.Logger.Warn(fmt.Sprintf("file %s started with Content-Length unknown, request headers: %v response headers: %v", fn, req.Header, resp.Header))
		}

		written, err := t.copy(f, resp.Body, d.bytesChan)

		resp.Body.Close()
		if written == resp.ContentLength || resp.ContentLength == -1 {
			f.Close()
			err := os.Rename(part, t.LocalPath)
			if err != nil {
				onErr("rename .part file", err)
				return
			}
			t.Status = Finished
			d.Logger.Info("task finished:", fn, "size:", strconv.FormatInt(written, 10))
			if t.AfterFinished != nil {
				// call AfterFinished hook
				t.AfterFinished(t)
			}
			return
		}
		d.Logger.Debug(fmt.Sprintf("ContentLength doesn't match, bytes written: %d, ContentLength: %d, url: %s, saving to: %s, request header: %v, response header: %v, error: %v", written, resp.ContentLength, req.URL, t.LocalPath, req.Header, resp.Header, err))
		if err := f.Sync(); err != nil {
			onErr("sync file", err)
			return
		}

		bytes += written
	}
}
