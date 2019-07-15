package client

import (
	"encoding/binary"
	"errors"
	"log"
	"net"
	"os/exec"
	"strings"
	"time"
)

var (
	host           = "cn.ntp.org.cn:123"    // ntp服务器
	driftThreshold = 500 * time.Millisecond // 允许的时间误差范围
)

type Packet struct {
	Settings       uint8  // leap yr indicator, ver number, and mode
	Stratum        uint8  // stratum of local clock
	Poll           int8   // poll exponent
	Precision      int8   // precision exponent
	RootDelay      uint32 // root delay
	RootDispersion uint32 // root dispersion
	ReferenceID    uint32 // reference id
	RefTimeSec     uint32 // reference timestamp sec
	RefTimeFrac    uint32 // reference timestamp fractional
	OrigTimeSec    uint32 // origin time secs
	OrigTimeFrac   uint32 // origin time fractional
	RxTimeSec      uint32 // receive time secs
	RxTimeFrac     uint32 // receive time frac
	TxTimeSec      uint32 // transmit time secs
	TxTimeFrac     uint32 // transmit time frac
}

// Client 同步时间并进行修改系统时间
func Client() error {
	conn, err := net.Dial("udp", host)
	if err != nil {
		log.Fatalf("udp dial errorL: %v", err)
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		log.Fatalf("set deadline error: %s", err)
		return err
	}
	req := &Packet{Settings: 0x1B}

	// 写入请求
	if err := binary.Write(conn, binary.BigEndian, req); err != nil {
		log.Fatalf("Write conn error: %s", err)
		return err
	}

	// 读取socket
	resp := &Packet{}
	if err := binary.Read(conn, binary.BigEndian, resp); err != nil {
		log.Fatalf("read socket error: %s", err)
		return err
	}

	/*
		Unix 时间是一个开始于 1970 年的纪元（或者说从 1970 年开始的秒数）。
		然而 NTP 使用的是另外一个纪元，从 1900 年开始的秒数。
		因此，从 NTP 服务端获取到的值要正确地转成 Unix 时间必须减掉这 70 年间的秒数 (1970-1900)
	*/
	sec := int64(resp.TxTimeSec)                                                          // 秒数
	frac := (int64(resp.TxTimeFrac) * 1e9) >> 32                                          // 纳秒位
	nanosec := sec*1e9 + frac                                                             // 纳秒时间戳
	tt := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nanosec)).Local() // 获得从1900年1月1日开始的纳秒时间戳
	standardTime := tt.Format("20060102 15:04:05.999999999")
	log.Println(standardTime)
	systemNow := time.Now()
	// 时间差
	diffTime := tt.Sub(systemNow)

	// 如果差值在允许的误差范围之内，则不用修改系统时间
	if diffTime > -driftThreshold && diffTime < driftThreshold {
		return nil
	}

	// 修改系统时间
	if err := modifySysTime(standardTime); err != nil {
		return err
	}
	return nil
}

// modifySysTime 修改系统时间(目前支持linux), 时间格式为("20060102 15:04:05.999999999")
func modifySysTime(standardTime string) error {
	if !IsRoot() { // 非root用户无法修改系统时间
		return errors.New("no permission to modify system time")
	}
	// 修改系统时间
	cmd := exec.Command("date", "-s", standardTime)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// IsRoot 是否为root用户
func IsRoot() bool {
	cmd := exec.Command("whoami") // 通过whoani来查看用户
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	err = cmd.Run()
	if err != nil {
		log.Fatalf("exec Run command error: %v", err)
		return false
	}
	if strings.Contains(string(out), "root") {
		return true
	}
	return false
}
