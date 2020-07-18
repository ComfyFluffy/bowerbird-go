package downloader

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/WOo0W/bowerbird/helper"
)

type taskStatus int

const (
	Pending taskStatus = iota
	Running
	Finished
	Paused
	Canceled
	Failed
)

const (
	defaultRetryMax     = 30
	defaultRetryWaitMin = 1 * time.Second
	defaultRetryWaitMax = 60 * time.Second
)

func filenameFromPath(path string, windowsSafe bool) string {
	b := filepath.Base(path)
	path = replacerAll.Replace(path)
	if windowsSafe {
		path = replacerOnWindows.Replace(path)
	}
	return b
}

type Backoff func(min, max time.Duration, tries int) time.Duration

type Task struct {
	bytesNow int64

	BytesLastSec int64
	Err          error
	Status       taskStatus
	Request      *http.Request
	LocalPath    string
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
	stop              chan struct{}
	globleBytesTicker *time.Ticker
	bytesChan         chan int64
	bytesNow,

	BytesLastSec int64

	in, out chan *Task

	Tasks []*Task
	Done  chan int

	tasksAdded, tasksDone int

	Client       *http.Client
	RetryMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	Backoff      Backoff
}

func (d *Downloader) worker() {
	for {
		select {
		case <-d.stop:
			return
		case t := <-d.in:
			d.Download(t)
			d.out <- t
		}
	}
}

func (d *Downloader) Start(workers int) {
	if d.runningWorkers > 0 {
		return
	}
	d.stop = make(chan struct{})
	d.Done = make(chan int)
	d.globleBytesTicker = time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case b := <-d.bytesChan:
				d.bytesNow += b
			case <-d.globleBytesTicker.C:
				d.BytesLastSec = d.bytesNow
				d.bytesNow = 0
			case <-d.stop:
				d.bytesNow = 0
				d.BytesLastSec = 0
			case <-d.out:
				d.tasksDone++
				if d.tasksAdded == d.tasksDone {
					d.Done <- d.tasksDone
				}
			}
		}
	}()
	for i := 0; i < workers; i++ {
		go d.worker()
	}
	d.runningWorkers = workers
}

func (d *Downloader) Stop() {
	close(d.stop)
	d.globleBytesTicker.Stop()
}

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
	d.Tasks = append(d.Tasks, task)
	d.tasksAdded++
	go func() {
		d.in <- task
	}()
}

func New() *Downloader {
	return &Downloader{
		RetryMax:     defaultRetryMax,
		RetryWaitMax: defaultRetryWaitMax,
		RetryWaitMin: defaultRetryWaitMin,
		Backoff:      helper.DefaultBackoff,
		Client:       http.DefaultClient,
		in:           make(chan *Task),
		out:          make(chan *Task),
		Tasks:        []*Task{},
	}
}

// type Pool struct {
// 	mu             sync.Mutex
// 	stop           chan bool
// 	taskChan       chan *Task
// 	runningWorkers int

// 	Tasks  []*Task
// 	Client *http.Client

// 	RetryMax     int
// 	RetryWaitMin time.Duration
// 	RetryWaitMax time.Duration
// 	Backoff      Backoff
// }

// func (pool *Pool) Start(workers int) {
// 	if pool.runningWorkers != 0 {
// 		return
// 	}
// 	pool.runningWorkers = workers
// 	for i := 0; i < workers; i++ {
// 		go pool.Worker()
// 	}
// 	go func() {
// 		for {
// 			select {
// 			case <-pool.stop:
// 				return
// 			}
// 		}
// 	}()
// }

// func (pool *Pool) Stop(workers int) {
// 	if workers == 0 {
// 		workers = pool.runningWorkers
// 	}
// 	for i := 0; i < workers+2; i++ {
// 		pool.stop <- true
// 	}
// }

func (d *Downloader) Download(t *Task) {
	onErr := func(err error) {
		t.Status = Failed
		t.Err = err
	}

	t.Status = Running
	ctx := t.Request.Context()
	req := t.Request.Clone(ctx)
	tries := 0
	bytes := int64(0)
	f, err := os.OpenFile(t.LocalPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		onErr(err)
		log.Println("Task ERR: os.Open:", err)
		return
	}
	defer f.Close()

	for {
		resp, err := d.Client.Do(req)
		if err != nil {
			onErr(err)
			log.Println("Task ERR: Do:", req.URL, resp.Header)
			return
		}
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			r, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				err = fmt.Errorf("http code %d with reading error: %w", resp.StatusCode, err)
			} else {
				err = fmt.Errorf("http code %d with message: %s", resp.StatusCode, r)
			}

			onErr(err)
			log.Println(err)
			return
		}

		tries++

		written, err := t.copy(f, resp.Body, d.bytesChan)

		resp.Body.Close()
		if written == resp.ContentLength {
			t.Status = Finished
			log.Println("Task Finished, bytes:", written)
			return
		}
		if err := f.Sync(); err != nil {
			onErr(err)
			log.Println("Task ERR: f.Sync()", err)
			return
		}
		log.Println("Task ERR: Copy:", err)
		if tries > d.RetryMax {
			onErr(err)
			log.Println("Max tries", tries, resp)
			return
		}
		select {
		case <-ctx.Done():
			log.Println("Task ctx.Done", ctx.Err(), t.Status)
			return
		case <-time.After(d.Backoff(d.RetryWaitMin, d.RetryWaitMax, tries)):
		}
		bytes += written
		req.Header["Range"] = []string{fmt.Sprintf("bytes=%d-", bytes)}
	}
}

// func (pool *Pool) Worker() {
// Loop:
// 	for {
// 		select {
// 		case <-pool.stop:
// 			log.Println("worker: exit")
// 			break Loop
// 		case t := <-pool.taskChan:
// 			log.Println("worker: task started", t)
// 			pool.Download(t)
// 			log.Println("worker: task finished", t)
// 		}
// 	}
// }

func main() {
	u := "https://i.pximg.net/img-original/img/2020/06/04/11/26/29/82078769_p0.jpg"
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header["Referer"] = []string{"https://www.pixiv.net"}

	d := New()
	d.Start(1)
	d.Add(&Task{Request: req, LocalPath: "dl"})

	// p := &Pool{
	// 	Backoff:      defaultBackoff,
	// 	RetryMax:     defaultRetryMax,
	// 	RetryWaitMax: defaultRetryWaitMax,
	// 	RetryWaitMin: defaultRetryWaitMin,
	// 	Tasks:        []*Task{},
	// 	Client:       &http.Client{},
	// 	taskChan:     make(chan *Task),
	// }

	// p.Tasks = append(p.Tasks, &Task{ctx: context.Background(), Req: req, SaveTo: partFilename(u)})
	// p.Start(1)

	select {}
}
