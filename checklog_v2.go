package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type Cfg struct {
	MonitorDir         string   `json:"monitorDir"`
	MonitorFilePattern string   `json:"monitorFilePattern"`
	Rules              []string `json:"rules"`
	EmailApiUrl        string   `json:"emailApiUrl"`
	EmailUserList      []string `json:"emailUserList"`
}

var Logs, LogsLast []string
var MonitorCfg Cfg

func CheckErr(e error) {
	if e != nil {
		return
	}
}

func AlarmMsg(msg, logFile string) string {
	hostname, _ := os.Hostname()
	addrs, _ := net.InterfaceAddrs()
	var IpAddr string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				IpAddr = ipnet.IP.String()
			}
		}
	}
	return "hostname: " + hostname + "\n" + "ip: " + IpAddr + "\n" + logFile + "\n" + msg
}

func Action(logFile, msg string) {
	sendMsg := AlarmMsg(msg, logFile)
	for _, mailAddr := range MonitorCfg.EmailUserList {
		SendMail(MonitorCfg.EmailApiUrl, mailAddr, sendMsg)
	}

	fmt.Println(logFile)
	s := strings.Split(logFile, "/")
	app := s[4]
	tomcatRestartCmd := "bash /home/lin/tomcat.sh " + app + " restart"
	cmd := exec.Command("/bin/bash", "-c", tomcatRestartCmd)
	var out bytes.Buffer

	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("%s", out.String())
}

func MonitorFileList(logfile, logFilePattern string) bool {
	b, _ := regexp.MatchString(logFilePattern, logfile)

	if b {
		return true
	} else {
		return false
	}
}

func CheckPattern(line, logPattern string) bool {
	b, _ := regexp.MatchString(logPattern, line)
	if b {
		return true
	} else {
		return false
	}
}

func listFile(folder, logFilePattern string) {
	files, _ := ioutil.ReadDir(folder)
	for _, file := range files {
		if file.IsDir() {
			listFile(folder+"/"+file.Name(), logFilePattern)
		} else {
			logfile := folder + "/" + file.Name()
			if MonitorFileList(logfile, logFilePattern) {
				Logs = append(Logs, logfile)
			}
		}
	}

}

func startTailFile(file string, rules []string) {

	var lastInode uint64
	var lastSeek int64
	lastInode = fileInode(file)
	for {
		lastSeek, lastInode = TailFile(file, lastInode, lastSeek, rules)
		time.Sleep(time.Second * 1)
	}
}

func fileInode(file string) uint64 {
	var statFile syscall.Stat_t
	if err := syscall.Stat(file, &statFile); err != nil {
		fmt.Println(err)
		return 0
	}
	return statFile.Ino
}

func TailFile(file string, lastInode uint64, lastSeek int64, rules []string) (int64, uint64) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		fmt.Println(err)
		return 0, 0
	}
	currentInode := fileInode(file)
	if currentInode != lastInode {
		lastSeek = 0
	}
	if lastSeek == 0 {
		f.Seek(lastSeek, 2)
	} else {
		f.Seek(lastSeek, 1)
	}

	fileReader := bufio.NewReader(f)

	for {
		inputString, readerError := fileReader.ReadString('\n')

		if readerError == io.EOF {
			currentOffset, _ := f.Seek(0, os.SEEK_CUR)
			return currentOffset, currentInode
		}

		for _, rule := range rules {
			if CheckPattern(inputString, rule) {
				fmt.Println(inputString)
				Action(file, inputString)
			}
		}
	}

}

func CheckLogChange(v string) bool {
	b := false
	for _, l := range LogsLast {
		if v == l {
			b = true
		}
	}
	if b {
		return false
	} else {
		return true
	}
}

func Rules(rule_url string) Cfg {
	u, _ := url.Parse(rule_url)
	q := u.Query()
	u.RawQuery = q.Encode()
	res, err := http.Get(u.String())
	if err != nil {
		fmt.Println(err)
	}
	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(result))
	var cfg Cfg
	json.Unmarshal(result, &cfg)
	return cfg
}

func SendMail(emailurl, user, msg string) {

	data := make(url.Values)
	data["subject"] = []string{"log monitor"}
	data["content"] = []string{msg}
	data["tos"] = []string{user}
	res, err := http.PostForm(emailurl, data)
	if err != nil {
		log.Println(err)
	}
	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Println(err)
		log.Printf("send email to  %s error.\n%s", msg)
	} else {
		log.Printf("send email to  %s success.\n%s", msg)
	}
	log.Println(string(result))

}

func main() {

	var cfgUrl string
	flag.StringVar(&cfgUrl, "i", "https://linyy.f3322.net/api/logmonitor", "config api url")
	flag.Parse()
	MonitorCfg = Rules(cfgUrl)

	folder := MonitorCfg.MonitorDir
	logFilePattern := MonitorCfg.MonitorFilePattern
	listFile(folder, logFilePattern)
	fmt.Println(Logs)

	rules := MonitorCfg.Rules

	for _, log := range Logs {
		go startTailFile(log, rules)
	}
	go func() {
		for {
			LogsLast = append(LogsLast, Logs...)
			Logs = []string{}
			listFile(folder, logFilePattern)
			for _, v := range Logs {
				if CheckLogChange(v) {
					go startTailFile(v, rules)
				}
			}
			time.Sleep(time.Second * 300)
		}

	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	s := <-c
	log.Println("Got signal:", s)
}
