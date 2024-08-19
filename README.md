# go-portScan封装


```go

// Package go_portScan -----------------------------
// @file      : main.go
// @author    : Xm17
// @contact   : https://github.com/kN6jq
// @time      : 2024/8/19 20:19
// -------------------------------------------
package main

import (
	"github.com/XinRoom/iprange"
	"github.com/kN6jq/go-portScan/core/host"
	"github.com/kN6jq/go-portScan/core/port"
	"github.com/kN6jq/go-portScan/core/port/tcp"
	"github.com/kN6jq/go-portScan/util"
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
	"sync"
	"time"
)

func main() {
	logger := util.NewLogger(util.DEBUG, "MyApp")

	single := make(chan struct{})
	retChan := make(chan port.OpenIpPort, 65535)
	go func() {
		for ret := range retChan {
			log.Println(ret)
		}
		single <- struct{}{}
	}()

	// 解析端口字符串并且优先发送 TopTcpPorts 中的端口, eg: 1-65535,top1000
	ports, err := port.ShuffleParseAndMergeTopPorts("top1000")
	if err != nil {
		log.Fatal(err)
	}

	// parse Ip
	it, _, _ := iprange.NewIter("127.0.0.1")

	// scanner
	ss, err := tcp.NewTcpScanner(retChan, tcp.DefaultTcpOption)
	if err != nil {
		logger.Fatal(err.Error())
	}

	// port scan func
	portScan := func(ip net.IP) {
		for _, _port := range ports { // port
			ss.WaitLimiter()
			ss.Scan(ip, _port)
		}
	}

	// Pool - ping and port scan
	var wgPing sync.WaitGroup
	poolPing, _ := ants.NewPoolWithFunc(50, func(ip interface{}) {
		_ip := ip.(net.IP)
		if host.IsLive(_ip.String(), true, 800*time.Millisecond) {
			portScan(_ip)
		}
		wgPing.Done()
	})
	defer poolPing.Release()

	start := time.Now()
	for i := uint64(0); i < it.TotalNum(); i++ { // ip索引
		ip := make(net.IP, len(it.GetIpByIndex(0)))
		copy(ip, it.GetIpByIndex(i)) // Note: dup copy []byte when concurrent (GetIpByIndex not to do dup copy)
		wgPing.Add(1)
		poolPing.Invoke(ip)
	}

	wgPing.Wait()
	ss.Close()
	<-single

	// cost
	log.Println(time.Since(start))
}


```