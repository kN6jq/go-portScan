package tcp

import (
	"context"
	"errors"
	"github.com/kN6jq/go-portScan/core/port"
	"github.com/kN6jq/go-portScan/core/service"
	limiter "golang.org/x/time/rate"
	"net"
	"sync"
	"time"
)

var DefaultTcpOption = port.ScannerOption{
	Rate:    1000,
	Timeout: 800,
}

type TcpScanner struct {
	ports   []uint16             // 指定端口
	retChan chan port.OpenIpPort // 返回值队列
	limiter *limiter.Limiter
	ctx     context.Context
	timeout time.Duration
	isDone  bool
	mu      sync.Mutex // 保护 isDone 和通道关闭状态
	wg      sync.WaitGroup
	option  port.ScannerOption
}

// NewTcpScanner Tcp扫描器
func NewTcpScanner(ctx context.Context, retChan chan port.OpenIpPort, option port.ScannerOption) (ts *TcpScanner, err error) {
	// option verify
	if option.Rate < 10 {
		err = errors.New("rate can not set < 10")
		return
	}
	if option.Timeout <= 0 {
		err = errors.New("timeout can not set to 0")
		return
	}

	ts = &TcpScanner{
		retChan: retChan,
		limiter: limiter.NewLimiter(limiter.Every(time.Second/time.Duration(option.Rate)), option.Rate/10),
		ctx:     ctx,
		timeout: time.Duration(option.Timeout) * time.Millisecond,
		option:  option,
	}

	return
}

// Scan 对指定IP和dst port进行扫描
func (ts *TcpScanner) Scan(ip net.IP, dst uint16) error {
	ts.mu.Lock()
	if ts.isDone {
		ts.mu.Unlock()
		return errors.New("scanner is closed")
	}
	ts.mu.Unlock()

	ts.wg.Add(1)
	go func() {
		defer ts.wg.Done()
		openIpPort := port.OpenIpPort{
			Ip:   ip,
			Port: dst,
		}

		// 处理错误
		var err error
		openIpPort, err = service.PortIdentify(ip, dst, ts.timeout)
		if err != nil {
			return
		}

		select {
		case ts.retChan <- openIpPort:
		case <-time.After(time.Second): // 超时
			return
		case <-ts.ctx.Done():
			return
		}
	}()
	return nil
}

func (ts *TcpScanner) Wait() {
	ts.wg.Wait()
}

// Close 关闭通道
func (ts *TcpScanner) Close() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.isDone {
		return
	}

	ts.isDone = true
	close(ts.retChan)
	ts.wg.Wait()
}

// WaitLimiter Waiting for the speed limit
func (ts *TcpScanner) WaitLimiter() error {
	return ts.limiter.Wait(ts.ctx)
}
