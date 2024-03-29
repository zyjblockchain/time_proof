package client

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	host           = "cn.ntp.org.cn:123"    // ntp服务器
	driftThreshold = 500 * time.Millisecond // 允许的时间误差范围
)

type durationSlice []time.Duration

func (s durationSlice) Len() int           { return len(s) }
func (s durationSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s durationSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

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
func Client(measurements int) error {
	diffs := make([]time.Duration, 0, measurements)

	for i := 0; i < measurements+2; i++ {
		conn, err := net.Dial("udp", host)
		if err != nil {
			log.Fatalf("udp dial errorL: %v", err)
			return err
		}

		send := time.Now()
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
		conn.Close()
		elapsed := time.Since(send)
		fmt.Println("传输时间：", elapsed.Nanoseconds())
		/*
			Unix 时间是一个开始于 1970 年的纪元（或者说从 1970 年开始的秒数）。
			然而 NTP 使用的是另外一个纪元，从 1900 年开始的秒数。
			因此，从 NTP 服务端获取到的值要正确地转成 Unix 时间必须减掉这 70 年间的秒数 (1970-1900)
		*/
		sec := int64(resp.TxTimeSec)                 // 秒数
		frac := (int64(resp.TxTimeFrac) * 1e9) >> 32 // 纳秒位
		fmt.Println("纳秒：", frac)
		nanosec := sec*1e9 + frac                                                             // 纳秒时间戳
		tt := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nanosec)).Local() // 获得从1900年1月1日开始的纳秒时间戳
		standardTime := tt.Format("20060102 15:04:05.999999999")
		log.Println("标准时间", standardTime)
		log.Println("系统时间", time.Now().Format("20060102 15:04:05.999999999"))

		// 时间差
		// diffTime := send.UnixNano() - tt.UnixNano() + elapsed.Nanoseconds() / 2
		diffTime := send.Sub(tt) + elapsed/2 // 与返回回来的时间差,本地时间 - 标准时间
		fmt.Println(diffTime)
		diffs = append(diffs, time.Duration(diffTime))
	}
	// 排序
	sort.Sort(durationSlice(diffs))
	// 去掉最高位和最低位求平均值
	var finalDiff time.Duration = 0
	temp := diffs[1]
	for i := 2; i < len(diffs)-1; i++ {
		next := temp + diffs[i]
		if temp^next < 0 { // 符号相反，说明溢出了
			finalDiff = diffs[1]
			break
		}
		temp = next
	}

	if finalDiff == time.Duration(0) {
		finalDiff = temp / time.Duration(measurements)
	}
	// 如果差值在允许的误差范围之内，则不用修改系统时间
	if finalDiff > -driftThreshold && finalDiff < driftThreshold {
		return nil
	}
	fmt.Println("相差时间22：", int64(finalDiff))
	// 修改系统时间
	if err := modifySysTime(int64(finalDiff)); err != nil {
		return err
	}
	return nil
}

// modifySysTime 传入参数为本地超过标准时间的时间戳数，可为正数和负数
func modifySysTime(overTimestamp int64) error {
	if runtime.GOOS != "linux" {
		return errors.New("currently only modify Linux system time is supported")
	}
	if !IsRoot() { // 非root用户无法修改系统时间
		return errors.New("no permission to modify system time")
	}

	// 修改系统时间
	standardTimestamp := time.Now().UnixNano() - overTimestamp // 计算出标准时间戳
	// 转换为系统设置时间的字符串类型
	standardTime := time.Unix(0, standardTimestamp).Local().Format("20060102 15:04:05.999999999")
	fmt.Println("待设置的时间：", standardTime)
	cmd := exec.Command("date", "-s", standardTime)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Fatalf("modify system time error: %v", err)
		return err
	}
	return nil
}

// IsRoot 是否为root用户
func IsRoot() bool {
	cmd := exec.Command("whoami") // 通过whoani来查看用户
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.CombinedOutput() error: %v", err)
		return false
	}
	// err = cmd.Run()
	// if err != nil {
	// 	log.Fatalf("exec Run command error: %v", err)
	// 	return false
	// }
	if strings.Contains(string(out), "root") {
		return true
	}
	return false
}
