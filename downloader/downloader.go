package downloader

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/helper"
)

type taskStatus int

//different status of a task
const (
	//Pending
	Pending taskStatus = iota
	//Running
	Running
	//Finished
	Finished
	Paused
	Canceled
	Failed
)

const (
	defaultRetryMax     = 30               //the default maximum number of retries
	defaultRetryWaitMin = 1 * time.Second  //the default minimum retry wait time
	defaultRetryWaitMax = 60 * time.Second //the default maximum retry wait time
)

//filenameFromPath get the name of the file from a path string
func filenameFromPath(path string, windowsSafe bool) string {
	b := filepath.Base(path)
	path = replacerAll.Replace(path)
	if windowsSafe {
		path = replacerOnWindows.Replace(path)
	}
	return b
}

//Backoff returns the wait time of a function
type Backoff func(min, max time.Duration, tries int) time.Duration

//Task stores information of a download task
type Task struct {
	bytesNow int64

	BytesLastSec int64
	Err          error
	Status       taskStatus
	Request      *http.Request
	LocalPath    string
	Overwrite    bool
}

func (t *Task) copy(dst io.Writer, src io.Reader, bytesChan chan int64) (written int64, err error) {
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

type Downloader struct {
	runningWorkers    int
	stopAll           chan struct{}
	globleBytesTicker *time.Ticker
	bytesChan         chan int64
	bytesNow,

	BytesLastSec int64

	Logger *log.Logger

	in, out chan *Task

	Tasks []*Task
	Done  chan int

	wg sync.WaitGroup

	Client       *http.Client
	TriesMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	Backoff      Backoff
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

func (d *Downloader) Start(workers int) {
	d.Logger.Debug(fmt.Sprintf("Starting downloader with %d workers", workers))
	if d.runningWorkers > 0 {
		return
	}
	d.globleBytesTicker = time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case b := <-d.bytesChan:
				d.bytesNow += b
			case <-d.globleBytesTicker.C:
				d.BytesLastSec = d.bytesNow
				d.bytesNow = 0
			case <-d.stopAll:
				d.bytesNow = 0
				d.BytesLastSec = 0
				return
				// case <-d.out:
			}
		}
	}()
	for i := 0; i < workers; i++ {
		go d.worker()
	}
	d.runningWorkers = workers
}

func (d *Downloader) Stop() {
	d.Logger.Debug("Stopping downloader")
	close(d.stopAll)
	d.globleBytesTicker.Stop()
}

// TODO: *Downloader.SetWorkers
// func (d *Downloader) SetWorkers(workers int) {
// 	if workers > d.runningWorkers {
// 		for i := 0; i < workers - d.runningWorkers; i++ {
// 			go d.worker()
// 		}
// 	} else {
// 		for i := 0; i < d.runningWorkers - workers; i++ {

// 		}
// 	}
// }

func (d *Downloader) Add(task *Task) {
	d.Logger.Debug("Adding *Task", task.Request.URL, task.LocalPath)
	d.Tasks = append(d.Tasks, task)
	d.wg.Add(1)
	go func() {
		d.in <- task
	}()
}

func (d *Downloader) Wait() {
	d.wg.Wait()
}

func New() *Downloader {
	return NewWithCliet(&http.Client{Transport: &http.Transport{}})
}

func NewWithCliet(c *http.Client) *Downloader {
	return &Downloader{
		TriesMax:     defaultRetryMax,
		RetryWaitMax: defaultRetryWaitMax,
		RetryWaitMin: defaultRetryWaitMin,
		Backoff:      helper.DefaultBackoff,
		Client:       c,
		Logger:       log.G,
		in:           make(chan *Task, 8192),
		out:          make(chan *Task),
		bytesChan:    make(chan int64),
		stopAll:      make(chan struct{}),
		Done:         make(chan int),
		Tasks:        []*Task{},
	}
}

func (d *Downloader) Download(t *Task) {
	if !t.Overwrite {
		if _, err := os.Stat(t.LocalPath); !os.IsNotExist(err) {
			d.Logger.Debug("file already exists:", t.LocalPath)
			return
		}
	}

	onErr := func(message string, err error) {
		d.Logger.Error(fmt.Sprintf("Task failed: download %s to %s: %s: %s", t.Request.URL, t.LocalPath, message, err))
		t.Status = Failed
		t.Err = err
	}

	t.Status = Running
	ctx := t.Request.Context()
	req := t.Request.Clone(ctx)
	tries := 0
	bytes := int64(0)
	part := t.LocalPath + ".part"
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
		bytes = fi.Size()
	}

	for {
		if bytes > 0 {
			req.Header["Range"] = []string{fmt.Sprintf("bytes=%d-", bytes)}
			d.Logger.Debug("Trying again with req.Header[\"Range\"] modified:", bytes)
		}

		tries++
		if tries > d.TriesMax {
			onErr("max tries", err)
			return
		}
		if tries > 1 {
			select {
			case <-ctx.Done():
				d.Logger.Debug("Task canceled by context:", ctx.Err())
				return
			case <-time.After(d.Backoff(d.RetryWaitMin, d.RetryWaitMax, tries)):
			}
		}

		resp, err := d.Client.Do(req)
		if err != nil {
			d.Logger.Debug("Response error:", err, req.URL)
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

			onErr("http code is not 2xx", err)
			return
		}

		written, err := t.copy(f, resp.Body, d.bytesChan)

		resp.Body.Close()
		if written == resp.ContentLength {
			t.Status = Finished
			d.Logger.Info("Task finished, filename:", filepath.Base(t.LocalPath))
			f.Close()
			err := os.Rename(part, t.LocalPath)
			if err != nil {
				onErr("rename .part file", err)
			}
			return
		}
		d.Logger.Debug(fmt.Sprintf("ContentLength doesn't match, bytes written: %d, url: %s, saving to: %s, error: %s", written, req.URL, t.LocalPath, err))
		if err := f.Sync(); err != nil {
			onErr("sync file", err)
			return
		}

		bytes += written
	}
}
