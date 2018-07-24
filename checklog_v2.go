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
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type Cfg struct {
	MonitorDir         string   `json:"monitorDir"`
	MonitorFilePattern string   `json:"monitorFilePattern"`
	Rules              []string `json:"rules"`
}

var Logs, LogsLast []string

func CheckErr(e error) {
	if e != nil {
		return
	}
}

func Action(logfile string) {
	fmt.Println(logfile)
	s := strings.Split(logfile, "/")
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
		time.Sleep(time.Second * 5)
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
				Action(file)
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
func main() {

	var cfgUrl string
	flag.StringVar(&cfgUrl, "i", "https://linyy.f3322.net/api/logmonitor", "config api url")
	flag.Parse()
	cfg := Rules(cfgUrl)

	folder := cfg.MonitorDir
	logFilePattern := cfg.MonitorFilePattern
	listFile(folder, logFilePattern)
	fmt.Println(Logs)

	rules := cfg.Rules

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

	time.Sleep(time.Hour * 1000000)
}

