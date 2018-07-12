package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func check1(ch1 chan bool, ip string, port string, timeOut, num int) {
	conn, err := net.DialTimeout("tcp", ip+":"+port, time.Duration(timeOut)*time.Second)
	if err != nil {
		log.Printf("第 %v 个 check fail.  %v ", num, err)
		ch1 <- false
	} else {
		time.Sleep(20 * time.Second)
		conn.Close()
		ch1 <- true
	}

}

func main() {
	arg := os.Args
	var ip, port string
	var timeOut, num int
	ch1 := make(chan bool)
	if len(arg) != 5 && len(arg) != 1 {
		fmt.Printf("Usage:%v {ip} {port} {timeout} {num} \n", arg[0])
		return
	}
	if len(arg) == 1 {
		ip = "www.baidu.com"
		port = "443"
		timeOut = 10
		num = 1000
	} else {
		ip = arg[1]
		port = arg[2]
		arg3 := arg[3]
		arg4 := arg[4]
		timeOut, _ = strconv.Atoi(arg3)
		num, _ = strconv.Atoi(arg4)
	}
	log.Printf("start check.......")
	log.Printf("address: %v    port:%v    timeout:%vs  count:%v \n", ip, port, timeOut, num)
	//ok, fail := check(ip, port, timeOut, num)
	for i := 1; i <= num; i++ {
		go check1(ch1, ip, port, timeOut, i)
	}
	log.Printf("gorouter end.......")
	ok := 0
	failed := 0
	for i := 1; i <= num; i++ {
		if <-ch1 {
			ok++
		} else {
			failed++
		}

	}
	log.Printf("end check, success:%v fail:%v \n", ok, failed)
}
