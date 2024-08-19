package service

import (
	"errors"
	"fmt"
	"github.com/kN6jq/go-portScan/core/port"
	"io"
	"net"
	"strings"
	"time"
)

// 优先级识别
func PortIdentify(ip net.IP, dst uint16, timeout time.Duration) (port.OpenIpPort, error) {
	var dwSvc int                 // dwSvc是返回的端口号
	var iRule = -1                // iRule是规则编号
	var bIsIdentification = false // bIsIdentification是标识符
	var packet []byte             // 报文
	var openIpPort port.OpenIpPort
	var err error
	//var iCntTimeOut = 0

	// 优先识别协议端口
	// 端口开放状态，发送报文，获取响应
	// 先判断端口是不是优先识别协议端口
	// todo: 自定义开启优先识别端口
	for _, svc := range St_Identification_Port {
		// 判断端口是否是优先识别协议端口
		if dst == svc.Port {
			bIsIdentification = true                       // 设置标识
			iRule = svc.Identification_RuleId              // 设置规则编号
			data := st_Identification_Packet[iRule].Packet // 根据规则编号获取报文
			// 发送报文
			openIpPort, dwSvc, err = SendIdentificationPacketFunction(data, ip, dst, timeout)
			if err != nil {
				break // 如果发送失败，则跳出循环
			}
			if dwSvc == SOCKET_CONNECT_FAILED {
				return openIpPort, errors.New("port Closed")
			}
			if (dwSvc > UNKNOWN_PORT && dwSvc <= SOCKET_CONNECT_FAILED) || dwSvc == SOCKET_READ_TIMEOUT {
				return openIpPort, nil
			}
			break
		}
	}

	// 识别其他协议
	// 发送其他协议查询包
	// 每个端口都会发送IdentificationProtocol的指纹报文识别
	for i := 0; i < iPacketMask; i++ {
		// 超时2次,不再识别
		if bIsIdentification && iRule == i {
			continue
		}
		if i == 0 {
			// 说明是http，数据需要拼装一下
			var szOption string
			if dst == 80 {
				szOption = fmt.Sprintf("%s%s\r\n\r\n", st_Identification_Packet[0].Packet, ip)
			} else {
				szOption = fmt.Sprintf("%s%s:%d\r\n\r\n", st_Identification_Packet[0].Packet, ip, dst)
			}
			packet = []byte(szOption)
		} else {
			packet = st_Identification_Packet[i].Packet
		}
		openIpPort, dwSvc, err = SendIdentificationPacketFunction(packet, ip, dst, timeout)
		if i == 0 {
		}
		if err != nil {
			break // 如果发送失败，则跳出循环
		}
		if dwSvc == SOCKET_CONNECT_FAILED {
			return openIpPort, errors.New("port Closed")
		}
		if (dwSvc > UNKNOWN_PORT && dwSvc <= SOCKET_CONNECT_FAILED) || dwSvc == SOCKET_READ_TIMEOUT {
			return openIpPort, nil
		}
	}
	return openIpPort, err
}

// 发送识别报文
func SendIdentificationPacketFunction(data []byte, ip net.IP, ports uint16, timeout time.Duration) (httpInfo port.OpenIpPort, dwSvcs int, err error) {
	even := port.OpenIpPort{
		Ip:       ip,
		Port:     ports,
		HttpInfo: &port.HttpInfo{},
	}

	addr := fmt.Sprintf("%s:%d", ip, ports)
	var dwSvc int = UNKNOWN_PORT
	conn, err := net.DialTimeout("tcp", addr, timeout) // todo 超时时间自定义
	if err != nil {
		dwSvc = SOCKET_CONNECT_FAILED
		return even, dwSvc, errors.New("port Closed")
	}
	defer conn.Close()
	if _, err := conn.Write(data); err != nil {
		return even, dwSvc, errors.New("write data error")
	}
	var fingerprint = make([]byte, 0, 65535)
	var tmp = make([]byte, 256)
	// 存储读取的字节数
	var num int
	var szBan string
	var szSvcName string
	readTimeout := 3 * time.Second
	conn.SetReadDeadline(time.Now().Add(readTimeout))

	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				dwSvc = SOCKET_READ_TIMEOUT
			}
			break
		}
		if n > 0 {
			num += n
			fingerprint = append(fingerprint, tmp[:n]...)
		} else {
			break
		}
	}
	if num > 0 {
		dwSvc = ComparePackets(fingerprint, num, &szBan, &szSvcName)
		if dwSvc > UNKNOWN_PORT && dwSvc < SOCKET_CONNECT_FAILED {
			if szSvcName == "ssl/tls" || szSvcName == "http" {
				var rst Result
				rst = GetHttpTitle(ip, HTTP, ports)
				if rst.StatusCode == 400 {
					rst = GetHttpTitle(ip, HTTPS, ports)
					szSvcName = "https"
				} else {
					szSvcName = "http"
				}
				if rst.Title != "" {
					even.HttpInfo.Title = rst.Title
				}
				if rst.StatusCode != 0 {
					even.HttpInfo.StatusCode = rst.StatusCode
				}
				if rst.Favicon != "" {
					even.HttpInfo.Favicon = rst.Favicon
				}
				if rst.URL != "" {
					even.HttpInfo.Url = rst.URL
				}
				if rst.ContentLength != 0 {
					even.HttpInfo.ContentLen = rst.ContentLength
				}
				if rst.Finger != "" {
					even.HttpInfo.Fingers = rst.Finger
				}
				for _, waf := range Waf_Title {
					if strings.Contains(rst.Title, waf) {
						return even, dwSvc, errors.New("port waf")
					}
				}
				for _, webserver := range Waf_WebServer {
					if strings.Contains(rst.WebServer, webserver) {
						return even, dwSvc, errors.New("port server web")
					}
				}
				cert, err0 := GetCert(ip, ports)
				if err0 != nil {
					cert = ""
				}
				even.HttpInfo.Cert = cert
			} else {
				even.Banner = strings.TrimSpace(szBan)
			}
			even.Service = szSvcName
		}
	}

	return even, dwSvc, nil
}
