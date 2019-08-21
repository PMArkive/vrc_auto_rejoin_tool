package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hpcloud/tail"
)

const WolrdLogPrefix = "[VRCFlowManagerVRC] Destination set: wrld_"
const location = "Asia/Tokyo"
const timeFormat = "2006.01.02 15:04:05"

var worldReg = regexp.MustCompile(`wrld_.+`)

type Instance struct {
	Time time.Time
	ID   string
}
type Instances []Instance

func (in Instances) Len() int {
	return len(in)
}

func (in Instances) Less(i, j int) bool {
	return in[i].Time.Before(in[j].Time)
}

func (in Instances) Swap(i, j int) {
	in[i], in[j] = in[j], in[i]
}

func moved(runAt time.Time, l string) bool {
	if l == "" {
		return false
	}

	if !strings.Contains(l, WolrdLogPrefix) {
		return false
	}

	loc, err := time.LoadLocation(location)
	if err != nil {
		loc = time.FixedZone(location, 9*60*60)
	}

	logTime, err := time.ParseInLocation(timeFormat, l[:19], loc)

	if err != nil {
		fmt.Println(l)
		panic(err)
	}

	if logTime.Before(runAt) {
		return false
	}

	return true
}

func lunch(instance Instance) {
	cmd := &exec.Cmd{
		Path:        os.Getenv("COMSPEC"),
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		SysProcAttr: &syscall.SysProcAttr{CmdLine: `/S /C start vrchat://launch?id=` + instance.ID},
	}

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func parseLatestInstance(logs string) Instance {
	latestI := Instance{}

	for _, line := range strings.Split(logs, "\n") {
		if line == "" {
			continue
		}

		if !strings.Contains(line, WolrdLogPrefix) {
			continue
		}

		logTime, err := time.ParseInLocation(timeFormat, line[:19], loc)
		if err != nil {
			fmt.Println(line)
			log.Fatal(err)
		}

		latestI = Instance{Time: logTime, ID: line}
	}
	return latestI
}

var loc *time.Location

func main() {
	loc, err := time.LoadLocation(location)
	if err != nil {
		loc = time.FixedZone(location, 9*60*60)
	}

	path := `C:\Users\bootjp\AppData\LocalLow\VRChat\VRChat\`
	latestLog := ""
	lock := sync.Mutex{}
	var history = Instances{}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})
	var filterd []os.FileInfo
	for _, v := range files {
		if strings.Contains(v.Name(), "output_log") {
			filterd = append(filterd, v)
		}
	}

	for _, v := range filterd {
		fmt.Println(v.Name(), v.ModTime().Format(timeFormat))
	}

	if len(filterd) > 0 {
		latestLog = filterd[0].Name()
	} else {
		log.Fatal("log file not found.")
	}

	startAt := time.Now().In(loc)
	fmt.Println("RUNNING START AT", startAt.Format(timeFormat))

	t, err := tail.TailFile(path+latestLog, tail.Config{
		Follow:    true,
		MustExist: true,
	})

	if err != nil {
		log.Fatal(err)
	}

	content, err := ioutil.ReadFile(latestLog)
	if err != nil {
		log.Fatal(err)
	}

	i := parseLatestInstance(string(content))
	fmt.Println(i)
	var msg *tail.Line
	var ok bool
	for true {
		msg, ok = <-t.Lines
		if !ok {
			continue
		}

		text := msg.Text
		// todo 起動時インスタンスの取得
		if moved(startAt, text) {
			lock.Lock()
			fmt.Println("instance move detect!!!")
			fmt.Println(text)
			if latestLog != text {
				latestLog = text
				history = append(history, Instance{Time: time.Time{}, ID: text})
				lunch(history[0])
			}
			lock.Unlock()
		}
	}
}
