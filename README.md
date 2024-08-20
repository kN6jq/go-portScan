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

func (s *sAssetTask) Scan(ctx context.Context, taskConfig *model.ScanTask) (err error) {
	g.Log().Info(ctx, "开始扫描任务")
	ipList := taskConfig.Tasktargets
	portList := taskConfig.Taskports
	timeout, err := time.ParseDuration(taskConfig.Tasktimeout)
	if err != nil {
		return fmt.Errorf("无效的超时设置: %v", err)
	}

	retChan := make(chan port.OpenIpPort, 65535)
	var wgResults sync.WaitGroup
	wgResults.Add(1)

	// 使用可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		defer wgResults.Done()
		g.Log().Info(ctx, "开始处理结果")
		for {
			select {
			case ret, ok := <-retChan:
				if !ok {
					g.Log().Info(ctx, "retChan已关闭，结果处理完成")
					return
				}
				// 处理结果...
				line := gtime.Now().Format("Y-m-d H:i:s") + " " + ret.Ip.String() + ":" + gconv.String(ret.Port) + " " + ret.Service + " " + ret.Banner +
					" " + ret.HttpInfo.Title + " " + gconv.String(ret.HttpInfo.StatusCode) + " " + ret.HttpInfo.Server + " " + ret.HttpInfo.TlsCN + " " + ret.HttpInfo.Url +
					" " + ret.HttpInfo.Favicon + " " + ret.HttpInfo.Fingers + " " + ret.HttpInfo.Cert
				g.Log().Info(ctx, line)
				data := &model.AssetServiceScanResult{
					IP:         ret.Ip.String(),
					Port:       int(ret.Port),
					Service:    ret.Service,
					Banner:     ret.Banner,
					Title:      ret.HttpInfo.Title,
					StatusCode: ret.HttpInfo.StatusCode,
					Server:     ret.HttpInfo.Server,
					TlsCN:      ret.HttpInfo.TlsCN,
					URL:        ret.HttpInfo.Url,
					Favicon:    ret.HttpInfo.Favicon,
					Fingers:    ret.HttpInfo.Fingers,
					Cert:       ret.HttpInfo.Cert,
					Time:       gtime.Now(),
				}
				service.AssetService().Save(ctx, data)
			case <-ctx.Done():
				g.Log().Info(ctx, "上下文已取消，结果处理goroutine退出")
				return
			}
		}
	}()

	g.Log().Info(ctx, "解析端口")
	ports, err := port.ShuffleParseAndMergeTopPorts(portList)
	if err != nil {
		g.Log().Error(ctx, "解析端口失败:", err.Error())
		cancel()
		wgResults.Wait()
		return err
	}

	g.Log().Info(ctx, "解析IP")
	it, _, _ := iprange.NewIter(ipList)

	g.Log().Info(ctx, "创建TcpScanner")
	ss, err := tcp.NewTcpScanner(ctx, retChan, tcp.DefaultTcpOption)
	if err != nil {
		g.Log().Error(ctx, "创建TcpScanner失败:", err.Error())
		cancel()
		wgResults.Wait()
		return err
	}

	var wgScan sync.WaitGroup
	defer func() {
		g.Log().Info(ctx, "等待所有扫描goroutine完成")
		wgScan.Wait()
		g.Log().Info(ctx, "所有扫描goroutine已完成")

		g.Log().Info(ctx, "开始关闭TcpScanner")
		ss.Close()
		g.Log().Info(ctx, "TcpScanner已关闭")

		cancel() // 取消上下文，通知结果处理 goroutine 退出
		g.Log().Info(ctx, "等待结果处理完成")
		wgResults.Wait()
		g.Log().Info(ctx, "结果处理已完成")
	}()

	portScan := func(ip net.IP) {
		defer wgScan.Done()
		for _, port := range ports {
			if err := ss.WaitLimiter(); err != nil {
				g.Log().Error(ctx, fmt.Sprintf("等待限速器失败: %v", err))
				return
			}
			select {
			case <-ctx.Done():
				g.Log().Info(ctx, "扫描被取消")
				return
			default:
				if err := ss.Scan(ip, port); err != nil {
					g.Log().Error(ctx, fmt.Sprintf("扫描 %s:%d 失败: %v", ip, port, err))
				}
			}
		}
	}

	g.Log().Info(ctx, "创建Ping和端口扫描池")
	var wgPing sync.WaitGroup
	poolPing, _ := ants.NewPoolWithFunc(50, func(ip interface{}) {
		defer wgPing.Done()
		_ip := ip.(net.IP)
		if host.IsLive(_ip.String(), true, timeout) {
			wgScan.Add(1)
			go portScan(_ip)
		}
	})
	defer poolPing.Release()

	start := time.Now()
	g.Log().Info(ctx, "开始扫描IP")
	for i := uint64(0); i < it.TotalNum(); i++ {
		ip := make(net.IP, len(it.GetIpByIndex(0)))
		copy(ip, it.GetIpByIndex(i))
		wgPing.Add(1)
		if err := poolPing.Invoke(ip); err != nil {
			g.Log().Error(ctx, fmt.Sprintf("调用Ping池失败: %v", err))
		}
	}

	g.Log().Info(ctx, "等待所有Ping完成")
	wgPing.Wait()
	g.Log().Info(ctx, "所有Ping已完成")

	g.Log().Info(ctx, "扫描完成")
	g.Log().Info(ctx, time.Since(start))
	return
}


```