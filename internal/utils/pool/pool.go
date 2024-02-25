package pool

import (
	"time"
)

type Pool struct {
	work chan func()
	sem  chan struct{}
}

func NewPool(size int) *Pool {
	return &Pool{
		work: make(chan func()),
		sem:  make(chan struct{}, size),
	}
}

func (p *Pool) Schedule(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.worker(task)
	}
}

func (p *Pool) worker(task func()) {
	for {
		task()
		select {
		case task = <-p.work:
		case <-time.After(time.Second * 5):
			<-p.sem
			return
		}
	}
}
