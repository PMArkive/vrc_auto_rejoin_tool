package vrcarjt

import (
	"errors"
	"fmt"

	"os/exec"
	"runtime"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/hpcloud/tail"
	"github.com/jinzhu/now"
	gops "github.com/mitchellh/go-ps"
	"github.com/shirou/gopsutil/process"

	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const WorldLogIdentifier = "] Destination set: wrld_"
const Location = "Local"
const TimeFormat = "2006.01.02 15:04:05"
const vrcRelativeLogPath = `\AppData\LocalLow\VRChat\VRChat\`
const Timeout = "Timeout: Your connection to VRChat timed out."

var BuildVersion = "v0.0.0"

func NewVRCAutoRejoinTool() *VRCAutoRejoinTool {
	conf := LoadConf("setting.yml")

	return &VRCAutoRejoinTool{
		Config:         conf,
		Args:           "",
		LatestInstance: Instance{},
		EnableRejoin:   !conf.EnableSleepDetector, // EnableSleepDetectorがOnのとき即座にインスタンス移動の検出をしないため
		InSleep:        false,
		rejoinLock:     &sync.Mutex{},
		playAudioLock:  &sync.Mutex{},
		running:        false,
		shutdown:       false,
	}
}

// VRCAutoRejoinTool ...
type VRCAutoRejoinTool struct {
	Config         *Setting
	Args           string
	LatestInstance Instance
	EnableRejoin   bool
	InSleep        bool
	rejoinLock     *sync.Mutex
	playAudioLock  *sync.Mutex
	running        bool
	shutdown       bool
}

type AutoRejoin interface {
	Run() error
	IsRun() bool
	ParseLatestInstance(path string) (Instance, error)
	SleepStart()
	Stop() error
	GetUserHome() string
}

func (v *VRCAutoRejoinTool) IsRun() bool {
	v.rejoinLock.Lock()
	defer v.rejoinLock.Unlock()
	return v.running
}
func (v *VRCAutoRejoinTool) IsShutdown() bool {
	v.rejoinLock.Lock()
	defer v.rejoinLock.Unlock()
	return v.shutdown
}

func (v *VRCAutoRejoinTool) SleepStart() {
	v.rejoinLock.Lock()
	defer v.rejoinLock.Unlock()
	v.InSleep = true
}

func (v *VRCAutoRejoinTool) Stop() error {
	if !v.running {
		return nil
	}
	v.rejoinLock.Lock()
	defer v.rejoinLock.Unlock()
	go v.playAudioFile("stop.wav")
	v.running = false

	return nil
}

func (v *VRCAutoRejoinTool) sleepInstanceDetector() Instance {
	return Instance{}
}

func (v *VRCAutoRejoinTool) GetUserHome() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func init() {
	var err error
	time.Local, err = time.LoadLocation(Location)
	if err != nil {
		time.Local = time.FixedZone(Location, 9*60*60)
	}
}

func (v *VRCAutoRejoinTool) Run() error {

	home := v.GetUserHome()

	if home == "" {
		return errors.New("home folder not found")
	}

	var err error
	v.Args, err = v.findProcessArgsByName("VRChat.exe")
	if err == ErrProcessNotFound {
		go v.playAudioFile("start_vrc.wav")
		v.rejoinLock.Lock()
		v.running = false
		v.rejoinLock.Unlock()
		return nil
	}
	if err != nil {
		v.rejoinLock.Lock()
		v.running = false
		v.rejoinLock.Unlock()
		return err
	}

	v.rejoinLock.Lock()
	v.running = true
	v.shutdown = false
	v.rejoinLock.Unlock()

	go v.playAudioFile("start.wav")
	path := home + vrcRelativeLogPath
	latestLog, err := v.fetchLatestLogName(path)
	if err != nil {
		return fmt.Errorf("log file not found. %s", err)
	}

	start := time.Now().In(time.Local)
	fmt.Println("RUNNING START AT", start.Format(TimeFormat))

	v.LatestInstance, err = v.ParseLatestInstance(path + latestLog)
	if err != nil {
		return err
	}

	t, err := tail.TailFile(path+latestLog, tail.Config{
		Follow:    true,
		MustExist: true,
		ReOpen:    true,
		Poll:      true,
	})
	if err != nil {
		v.rejoinLock.Lock()
		v.running = false
		v.rejoinLock.Unlock()
		return err
	}
	if v.Config.EnableProcessCheck {
		go v.processWatcher()
	}
	go v.logInspector(t, start)

	return nil
}

func (v *VRCAutoRejoinTool) rejoin(i Instance, killProcess bool) error {
	v.rejoinLock.Lock()
	v.shutdown = true
	defer func() {
		v.running = false
		v.rejoinLock.Unlock()
	}()
	if killProcess {
		err := v.killProcessByName("VRChat.exe")
		if err != nil {
			log.Println(err)
		}
	}

	args := prepareExecArgs(v.Args, i)
	cmd := exec.Command(args.ExePath, args.Args...)

	return cmd.Start()
}

func (v *VRCAutoRejoinTool) ParseLatestInstance(path string) (Instance, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		return Instance{}, err
	}

	return v.parseLatestInstance(string(content))
}

// ErrProcessNotFound is an error that is returned when the target process could not be found
var ErrProcessNotFound = errors.New("process not found")

func (v *VRCAutoRejoinTool) findProcessPIDByName(name string) (int32, error) {
	processes, err := gops.Processes()
	if err != nil {
		return -1, err
	}

	for _, p := range processes {
		if strings.Contains(p.Executable(), name) {
			return int32(p.Pid()), nil
		}
	}

	return -1, ErrProcessNotFound
}

func (v *VRCAutoRejoinTool) findProcessArgsByName(name string) (string, error) {
	pid, err := v.findProcessPIDByName(name)
	if err != nil {
		return "", ErrProcessNotFound
	}

	p, err := process.NewProcess(pid)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return p.Cmdline()
}

func (v *VRCAutoRejoinTool) killProcessByName(name string) error {
	pid, err := v.findProcessPIDByName(name)
	if err != nil {
		return err
	}

	p, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	return p.Kill()
}

func (v *VRCAutoRejoinTool) inTimeRange(start time.Time, end time.Time, target time.Time) bool {
	// https://stackoverflow.com/questions/55093676/checking-if-current-time-is-in-a-given-interval-golang
	if start.Before(end) {
		return !target.Before(start) && !target.After(end)
	}
	if start.Equal(end) {
		return target.Equal(start)
	}
	return !start.After(target) || !end.Before(target)
}

func (v *VRCAutoRejoinTool) playAudioFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = streamer.Close()
	}()

	v.playAudioLock.Lock()
	defer v.playAudioLock.Unlock()
	wait := &sync.WaitGroup{}
	wait.Add(1)
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		log.Fatal(err)
	}

	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		wait.Done()
	})))
	wait.Wait()
}

func (v *VRCAutoRejoinTool) parseLatestInstance(s string) (Instance, error) {
	latestInstance := Instance{}

	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			continue
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if !strings.Contains(line, WorldLogIdentifier) {
			continue

		}

		instance, err := NewInstanceByLog(line)
		if err != nil {
			return instance, err
		}
		latestInstance = instance
	}
	return latestInstance, nil
}
func (v *VRCAutoRejoinTool) fetchLatestLogName(path string) (string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println(err)
		return "", err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})
	var filtered []os.FileInfo
	for _, v := range files {
		if strings.Contains(v.Name(), "output_log") {
			filtered = append(filtered, v)
		}
	}

	latestLog := ""
	if len(filtered) > 0 {
		latestLog = filtered[0].Name()
	}

	return latestLog, nil
}

func (v *VRCAutoRejoinTool) processWatcher() {

	for v.IsRun() && !v.IsShutdown() {
		log.Println("process watcher available")
		_, err := v.findProcessPIDByName("VRChat.exe")
		if err == ErrProcessNotFound {
			if v.Config.EnableRejoinNotice {
				go v.playAudioFile("rejoin_notice.wav")
				time.Sleep(1 * time.Minute)
			}
			// 警告オーディオ再生中に止まった場合なにもしない
			if !v.running {
				log.Println("cancel rejoin")
				v.shutdown = true
				return
			}
			err := v.rejoin(v.LatestInstance, false)
			if err != nil {
				log.Println(err)
			}
			return
		}
		time.Sleep(10 * time.Second)
	}
	log.Println("process watcher clean up by other.")

}

func (v *VRCAutoRejoinTool) logInspector(tail *tail.Tail, at time.Time) {

	for msg := range tail.Lines {
		if !v.IsRun() || v.IsShutdown() {
			log.Println("log watcher clean up by other.")
			tail.Cleanup()
			break
		}

		logLine := msg.Text

		if !v.isMove(at, logLine) && !v.isTimeout(logLine) {
			continue
		}

		log.Println("instance move detected")

		if v.Config.EnableRadioExercises {
			start, err := now.ParseInLocation(time.Local, "05:45")
			if err != nil {
				log.Println(err)
				continue
			}

			end, err := now.ParseInLocation(time.Local, "08:00")
			if err != nil {
				log.Println(err)
				continue
			}

			if v.inTimeRange(start, end, time.Now().In(time.Local)) {
				continue
			}
		}

		if v.Config.EnableRejoinNotice {
			go v.playAudioFile("rejoin_notice.wav")
			time.Sleep(1 * time.Minute)
		}

		// 警告オーディオ再生中に止まった場合なにもしない
		if !v.running {
			tail.Cleanup()
			log.Println("cancel rejoin")
			return
		}

		err := v.rejoin(v.LatestInstance, true)
		if err != nil {
			log.Println(err)
		}

		tail.Cleanup()
		return
	}
}

func (v *VRCAutoRejoinTool) isMove(at time.Time, l string) bool {
	if l == "" {
		return false
	}

	if !strings.Contains(l, WorldLogIdentifier) {
		return false
	}

	i, err := NewInstanceByLog(l)
	if err != nil {
		return false
	}
	if i.Time.Before(at) {
		return false
	}

	if v.LatestInstance.ID == i.ID {
		return false
	}

	return true

}

func (v *VRCAutoRejoinTool) isTimeout(log string) bool {
	return strings.Contains(log, Timeout)
}

type Exec struct {
	ExePath string
	Args    []string
}

func prepareExecArgs(processArgs string, i Instance) Exec {
	// 既存の起動引数を用いて rejoin するインスタンスを指定する
	// TODO 既存の引数にインスタンス指定があれば取り除く
	args := processArgs + ` vrchat://launch?id=` + i.ID

	// 今動いている VRChat.exe までのパスを取得する
	// go の windows の exec は exe までのパスと引数を完全に別物として扱うため
	arg := strings.Split(args, `VRChat.exe`)
	exe := arg[:1][0] + `VRChat.exe`

	// C:\Program Files (x86) などのスペースを含む階層以下にある場合のVRChat.exe のパスは "" で囲まれているため除去する
	// 末尾の " は exe の組立時に VRChat.exe で追加しているため除去不要
	if strings.HasPrefix(exe, `"`) {
		exe = exe[1:]
	}

	tmpArgs := arg[1:][0]

	// C:\Program Files (x86) 以下の階層にある場合はexeのパスの " がのこるので除去する
	if strings.HasPrefix(tmpArgs, `"`) {
		tmpArgs = tmpArgs[1:]
	}

	exeArgs := strings.Fields(strings.TrimSpace(tmpArgs))

	return Exec{
		ExePath: exe,
		Args:    exeArgs,
	}
}
