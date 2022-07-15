package utils

import (
	"runtime"
	"time"
)

func LoadCPU(sec int) {
	done := make(chan int)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}

	time.Sleep(time.Second * time.Duration(sec))
	close(done)
}
