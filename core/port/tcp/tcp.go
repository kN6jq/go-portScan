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

// TcpScanner 结构体更新
type TcpScanner struct {
	ports   []uint16
	retChan chan port.OpenIpPort
	limiter *limiter.Limiter
	ctx     context.Context
	cancel  context.CancelFunc // 新增：用于取消操作的函数
	timeout time.Duration
	isDone  bool
	mu      sync.Mutex
	wg      sync.WaitGroup
	option  port.ScannerOption
}

// NewTcpScanner 函数更新
func NewTcpScanner(ctx context.Context, retChan chan port.OpenIpPort, option port.ScannerOption) (*TcpScanner, error) {
	if option.Rate < 10 {
		return nil, errors.New("rate cannot be set < 10")
	}
	if option.Timeout <= 0 {
		return nil, errors.New("timeout cannot be set to 0")
	}

	ctx, cancel := context.WithCancel(ctx)
	ts := &TcpScanner{
		retChan: retChan,
		limiter: limiter.NewLimiter(limiter.Every(time.Second/time.Duration(option.Rate)), option.Rate/10),
		ctx:     ctx,
		cancel:  cancel,
		timeout: time.Duration(option.Timeout) * time.Millisecond,
		option:  option,
	}

	return ts, nil
}

// Scan 方法更新
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

		var err error
		openIpPort, err = service.PortIdentify(ip, dst, ts.timeout)
		if err != nil {
			return
		}

		select {
		case ts.retChan <- openIpPort:
		case <-ts.ctx.Done():
			return
		}
	}()
	return nil
}

// Close 方法更新
func (ts *TcpScanner) Close() {
	ts.mu.Lock()
	if ts.isDone {
		ts.mu.Unlock()
		return
	}
	ts.isDone = true
	ts.cancel() // 取消所有正在进行的操作
	ts.mu.Unlock()

	ts.wg.Wait() // 等待所有 goroutine 完成
}

// 其他方法保持不变
func (ts *TcpScanner) Wait() {
	ts.wg.Wait()
}

func (ts *TcpScanner) WaitLimiter() error {
	return ts.limiter.Wait(ts.ctx)
}
