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
type Backoff func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration

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

//The core of downloader
type Downloader struct {
	runningWorkers    int
	stopAll           chan struct{}
	globleBytesTicker *time.Ticker
	bytesChan         chan int64
	bytesNow,
	//bytes downloaded in the last second
	BytesLastSec int64
	//logger of the downloader
	Logger *log.Logger

	notfull chan struct{}
	in      chan *Task
	//tasks that are to be downloaded
	Tasks []*Task
	//A finishing indicator
	Done chan int

	wg   sync.WaitGroup
	once sync.Once
	//http client for this downloader
	Client *http.Client
	//max number of retries
	TriesMax int
	//minimum wait time of retries
	RetryWaitMin time.Duration
	//maximum wait time of retries
	RetryWaitMax time.Duration
	//backoff function for retries
	Backoff Backoff
	//maximum threads for a downloader task
	MaxWorkers int
}

func (d *Downloader) worker() {
	for {
		select {
		case <-d.stopAll:
			return
		case t := <-d.in:
			d.Download(t)
			d.wg.Done()
			if len(d.notfull) == 0 {
				d.notfull <- struct{}{}
			}
			// d.out <- t
		}
	}
}

//Start method activates the downloader object
func (d *Downloader) Start() {
	d.once.Do(func() {
		d.notfull <- struct{}{}
		d.Logger.Debug(fmt.Sprintf("starting downloader"))

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

//Stop terminates the downloader object
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
//Add adds tasks to the downloader
func (d *Downloader) Add(task *Task) {
	// d.Logger.Debug("Adding *Task", task.Request.URL, task.LocalPath)
	d.Tasks = append(d.Tasks, task)
	d.wg.Add(1)

	go func() {
		select {
		case <-d.notfull:
			if len(d.in) < 8-1 {
				d.notfull <- struct{}{}
			}
			d.in <- task
		}

	}()
}

//Wait suspends the downloader
func (d *Downloader) Wait() {
	d.wg.Wait()
}

//NewWithDefaultClient returns a downloader with default client
func NewWithDefaultClient() *Downloader {
	return NewWithCliet(&http.Client{Transport: &http.Transport{}})
}

//NewWithCliet takes a http clinet and return a downloader with that client
func NewWithCliet(c *http.Client) *Downloader {
	return &Downloader{
		TriesMax:     defaultRetryMax,
		RetryWaitMax: defaultRetryWaitMax,
		RetryWaitMin: defaultRetryWaitMin,
		Backoff:      helper.DefaultBackoff,
		Client:       c,
		Logger:       log.G,
		in:           make(chan *Task, 8),
		bytesChan:    make(chan int64),
		stopAll:      make(chan struct{}),
		Done:         make(chan int),
		Tasks:        []*Task{},
		MaxWorkers:   8,
		notfull:      make(chan struct{}, 1),
	}
}

//Download method does the task
func (d *Downloader) Download(t *Task) {
	d.Client.Transport = &http.Transport{}
	//skip finished tasks
	if !t.Overwrite {
		if _, err := os.Stat(t.LocalPath); !os.IsNotExist(err) {
			// d.Logger.Debug("file already exists:", t.LocalPath)
			return
		}
	}
	log.G.Debug("starting task", t.Request.URL, t.LocalPath)

	onErr := func(message string, err error) {
		d.Logger.Error(fmt.Sprintf("task failed: download %s to %s: %s: %s", t.Request.URL, t.LocalPath, message, err))
		t.Status = Failed
		t.Err = err
	}
	//set task status to running
	t.Status = Running
	ctx := t.Request.Context()
	//get task request
	req := t.Request.Clone(ctx)
	//tries time
	tries := 0
	//?
	bytes := int64(0)
	//file name for partially downloaded files
	part := t.LocalPath + ".part"
	//make directory for the file
	os.MkdirAll(filepath.Dir(t.LocalPath), 0755)
	//open target file
	f, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		onErr("open file", err)
		return
	}
	defer f.Close()
	//get file status
	fi, err := f.Stat()
	if err != nil {
		d.Logger.Error("stat file", part, err)
	} else {
		bytes = fi.Size()
	}

	for {
		//slices?
		if bytes > 0 {
			req.Header["Range"] = []string{fmt.Sprintf("bytes=%d-", bytes)}
			d.Logger.Debug("Trying again with req.Header[\"Range\"] modified:", bytes)
		}

		tries++
		if tries > d.TriesMax {
			onErr("max tries", err)
			return
		}
		//?
		if tries > 1 {
			select {
			case <-ctx.Done():
				d.Logger.Debug("Task canceled by context:", ctx.Err())
				return
			case <-time.After(d.Backoff(d.RetryWaitMin, d.RetryWaitMax, tries, nil)):
			}
		}
		//send request from the task and get response
		resp, err := d.Client.Do(req)
		if err != nil {
			d.Logger.Debug("Response error:", err, req.URL)
			continue
		}
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			//deal with error response
			r, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				err = fmt.Errorf("http code %d with reading error: %w", resp.StatusCode, err)
			} else {
				err = fmt.Errorf("http code %d with message: %s", resp.StatusCode, r)
			}

			onErr("http code not ok", err)
			return
		}
		//get response content
		written, err := t.copy(f, resp.Body, d.bytesChan)

		resp.Body.Close()
		if written == resp.ContentLength { //do when finished
			t.Status = Finished
			d.Logger.Info("task finished:", filepath.Base(t.LocalPath))
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
