package main

import (
	"bytes"
	"fmt"
	"github.com/hpcloud/tail"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

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

func CheckTail(log string, rules []string) {
	t, err := tail.TailFile(log, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Location: &tail.SeekInfo{Offset: 0, Whence: 2},
	})

	CheckErr(err)
	for line := range t.Lines {
		for _, rule := range rules {
			if CheckPattern(line.Text, rule) {
				fmt.Println(line.Text)
				Action(log)
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

func main() {

	folder := "/home/lin/test"
	logFilePattern := "^/home/lin/test/[^ ]*/catalina.out$"
	listFile(folder, logFilePattern)
	fmt.Println(Logs)

	rules := []string{"OutOfMemoryError", "test"}

	for _, log := range Logs {
		go CheckTail(log, rules)
	}
	go func() {
		for {
			LogsLast = append(LogsLast, Logs...)
			Logs = []string{}
			listFile(folder, logFilePattern)
			for _, v := range Logs {
				if CheckLogChange(v) {
					go CheckTail(v, rules)
				}
			}
			time.Sleep(time.Second * 300)
		}

	}()

	time.Sleep(time.Hour * 1000000)
}
