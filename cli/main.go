package main

import (
	"fyne.io/fyne/dialog"
	"log"
	"net/url"
	"time"

	vrcarjt "github.com/bootjp/vrc_auto_rejoin_tool"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
)

var logo = canvas.NewImageFromFile("./logo.png")

func parseURL(urlStr string) *url.URL {
	link, err := url.Parse(urlStr)
	if err != nil {
		fyne.LogError("Could not parse URL", err)
	}

	return link
}

func help(a fyne.App, vrc *vrcarjt.VRCAutoRejoinTool) fyne.CanvasObject {
	logo.SetMinSize(fyne.NewSize(300, 300))
	return widget.NewVBox(
		layout.NewSpacer(),
		widget.NewHBox(layout.NewSpacer(), logo, layout.NewSpacer()),
		widget.NewHBox(layout.NewSpacer(),
			widget.NewHyperlink("BOOTH", parseURL("https://bootjp.booth.pm/items/1542381")),
			widget.NewLabel("-"),
			widget.NewHyperlink("GitHub", parseURL("https://github.com/bootjp/vrc_auto_rejoin_tool")),
			layout.NewSpacer(),
		),

		fyne.NewContainerWithLayout(layout.NewCenterLayout(),
			widget.NewTextGridFromString("version: v.X.X.X"),
		),
	)
}

func welcomeScreen(a fyne.App, v vrcarjt.AutoRejoin, w fyne.Window) fyne.CanvasObject {
	logo.SetMinSize(fyne.NewSize(250, 250))
	status := widget.NewTextGridFromString("Status: Stop")
	statusContainer := fyne.NewContainerWithLayout(layout.NewCenterLayout(),
		status,
	)
	start := widget.NewButton("Start", func() {
		if v.IsRun() {
			return
		}
		if err := v.Run(); err != nil {
			fyne.LogError(err.Error(), err)
			a.Quit()
		}

	})
	stop := widget.NewButton("Stop", func() {
		if !v.IsRun() {
			return
		}
		if err := v.Stop(); err != nil {
			fyne.LogError(err.Error(), err)
			a.Quit()
		}
	})
	// check status
	go func() {
		for {
			switch v.IsRun() {
			case true:
				status.SetText("Status: Running")
				start.Hidden = true
				stop.Hidden = false
			case false:
				status.SetText("Status: Stop")
				start.Hidden = false
				stop.Hidden = true
			}
			time.Sleep(1 * time.Second)
		}

	}()

	return widget.NewVBox(
		layout.NewSpacer(),
		widget.NewHBox(layout.NewSpacer(),
			widget.NewHyperlink("BOOTH", parseURL("https://bootjp.booth.pm/items/1542381")),
			widget.NewLabel("-"),
			widget.NewHyperlink("GitHub", parseURL("https://github.com/bootjp/vrc_auto_rejoin_tool")),
			layout.NewSpacer(),
		),
		widget.NewHBox(layout.NewSpacer(), logo, layout.NewSpacer()),
		statusContainer,
		//fyne.NewContainerWithLayout(layout.NewCenterLayout(),
		//	widget.NewTextGridFromString("version: v.X.X.X"),
		//),

		widget.NewGroup("Controls",
			fyne.NewContainerWithLayout(layout.NewGridLayout(1),
				start,
				stop,
			),
		),
	)

}

func settingScreen(a fyne.App, vrc *vrcarjt.VRCAutoRejoinTool, w fyne.Window) fyne.CanvasObject {

	pcheck := widget.NewCheck("enable_process_check", func(value bool) {
		log.Println("Check set to", value)
	})
	debug := widget.NewCheck("debug", func(value bool) {
		log.Println("Check set to", value)
	})
	radioex := widget.NewCheck("enable_radio_exercises", func(value bool) {
		log.Println("Check set to", value)
	})
	// enable_rejoin_notice
	notice := widget.NewCheck("enable_rejoin_notice", func(value bool) {
		log.Println("Check set to", value)
	})
	var (
		selectedfiles fyne.URIReadCloser
		fileerror     error
	)
	// Avoidance of errors until the release of the configuration screen
	_ = fileerror
	_ = selectedfiles
	return widget.NewVBox(
		layout.NewSpacer(),
		widget.NewHBox(pcheck),
		widget.NewHBox(debug),
		widget.NewHBox(radioex),
		widget.NewHBox(notice),
		layout.NewSpacer(),
		widget.NewGroup("",
			fyne.NewContainerWithLayout(layout.NewGridLayout(2),
				widget.NewButton("Save", func() {
					//v.SaveSetting()
				}),
				widget.NewButton("Load Setting", func() {
					dialog.ShowFileOpen(func(file fyne.URIReadCloser, err error) {
						selectedfiles = file
						err = fileerror
					}, w)
				}),
			),
		),
	)

}

const lockfile = "vrc_auto_rejoin_tool.rejoinLock"

func main() {
	vrc := vrcarjt.NewVRCAutoRejoinTool()

	home := vrc.GetUserHome()
	lock := vrcarjt.NewDupRunLock(home + `\AppData\Local\Temp\` + lockfile)
	ok, err := lock.Try()

	if err != nil || !ok {
		log.Println("auto rejoin tool が多重起動しています．")
	}

	if err = lock.Lock(); err != nil {
		log.Println("auto rejoin tool が多重起動しています．")
	}

	defer lock.UnLock()

	a := app.NewWithID("vrc_auto_rejoin_tool")
	a.SetIcon(logo.Resource)

	w := a.NewWindow("vrc_auto_rejoin_tool")
	tabs := widget.NewTabContainer(
		widget.NewTabItemWithIcon("Control", logo.Resource, welcomeScreen(a, vrc, w)),
		//widget.NewTabItemWithIcon("Setting", logo.Resource, settingScreen(a, vrc, w)),
	)
	w.SetContent(tabs)
	w.ShowAndRun()

}
