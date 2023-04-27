package miner

import (
	"fmt"
	"sync"
	"time"
)

func TimerTest() {
	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	wg.Add(2)

	timer := time.NewTimer(time.Second * 2)
	go func(t *time.Timer) {
		defer wg.Done()
		for {
			select {
			case <-t.C:
				fmt.Println("timer..")

				//Reset可以实现timer每隔固定时间段触发。reset 重新开始计时，如果调用时 t 还在等待中会返回真；如果 t已经到期或者被停止了会返回假。
				// t.Stop()  里面stop没用
				t.Reset(1 * time.Second)

			case <-stopChan:
				fmt.Println("timer goroutine stop.. ")
				return
			}

		}
	}(timer)

	ticker := time.NewTicker(time.Second * 1)
	go func(t *time.Ticker) {
		defer wg.Done()
		for {
			select {
			case <-t.C:
				fmt.Println("ticker", time.Now().Format("2006-01-02 15:04:05"))
			case <-stopChan:
				fmt.Println("ticker goroutine stop.. ")
				return
			}
		}
	}(ticker)

	time.Sleep(5 * time.Second)
	ticker.Stop()
	timer.Stop()
	stopChan <- struct{}{}
	stopChan <- struct{}{}

	defer func() {
		fmt.Println("stop")
	}()

	wg.Wait()

}
