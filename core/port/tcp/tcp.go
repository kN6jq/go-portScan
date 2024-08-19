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
	option  port.ScannerOption
	wg      sync.WaitGroup
}

// NewTcpScanner Tcp扫描器
func NewTcpScanner(retChan chan port.OpenIpPort, option port.ScannerOption) (ts *TcpScanner, err error) {
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
		ctx:     context.Background(),
		timeout: time.Duration(option.Timeout) * time.Millisecond,
		option:  option,
	}

	return
}

// Scan 对指定IP和dis port进行扫描
func (ts *TcpScanner) Scan(ip net.IP, dst uint16) error {
	if ts.isDone {
		return errors.New("scanner is closed")
	}
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

		ts.retChan <- openIpPort
	}()
	return nil
}

func (ts *TcpScanner) Wait() {
	ts.wg.Wait()
}

// Close chan
func (ts *TcpScanner) Close() {
	ts.isDone = true
	close(ts.retChan)
}

// WaitLimiter Waiting for the speed limit
func (ts *TcpScanner) WaitLimiter() error {
	return ts.limiter.Wait(ts.ctx)
}
