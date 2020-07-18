package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func filenameFromPath(path string) string {
	b := filepath.Base(path)

	if strings.ContainsAny(b, "/\\") {
		var uuid [16]byte

		io.ReadFull(rand.Reader, uuid[:])
		uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
		uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

		return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
	}

	// TODO: windows path replace
	// if runtime.GOOS == "windows" {
	// 	strings.NewReplacer("")
	// }
	return b
}

type Backoff func(min, max time.Duration, tries int) time.Duration

func defaultBackoff(min, max time.Duration, tries int) time.Duration {
	if sleep := (1 << tries) * min; sleep < max && sleep != 0 {
		return sleep
	}
	return max
}

type Task struct {
	ctx      context.Context
	bytesNow int64

	BytesLastSec int64
	Err          error
	Status       taskStatus
	Request      *http.Request
	SaveTo       string
}

func (t *Task) copy(dst io.Writer, src io.Reader) (written int64, err error) {
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
				written += int64(nw)

				select {
				case <-bytesTicker.C:
					t.BytesLastSec = t.bytesNow
					t.bytesNow = 0
				default:
					t.bytesNow += int64(nw)
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
	runningWorkers int
	stop           chan struct{}

	in, out chan *Task

	Tasks []*Task

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
		}
	}
}

func (d *Downloader) Start(workers int) {
	d.stop = make(chan struct{})
	for i := 0; i < workers; i++ {
		go d.worker()
	}
	d.runningWorkers = workers
}

func (d *Downloader) Stop() {
	close(d.stop)
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
	go func() {
		d.in <- task
	}()
}

func New() *Downloader {
	return &Downloader{
		RetryMax:     defaultRetryMax,
		RetryWaitMax: defaultRetryWaitMax,
		RetryWaitMin: defaultRetryWaitMin,
		Backoff:      defaultBackoff,
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
	req := t.Request.Clone(t.ctx)
	tries := 0
	bytes := int64(0)
	f, err := os.OpenFile(filepath.Join(t.SaveTo, filenameFromPath(req.URL.Path)), os.O_CREATE|os.O_APPEND, 0644)
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

		tries++

		written, err := t.copy(f, resp.Body)

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
		case <-t.ctx.Done():
			log.Println("Task ctx.Done", t.ctx.Err(), t.Status)
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
	d.Add(&Task{ctx: context.Background(), Request: req, SaveTo: "dl"})

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
