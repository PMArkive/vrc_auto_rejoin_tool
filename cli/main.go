//Go source to generate custom bundled-fonts.go
//Replace...)\fyne-io\fyne\theme\bundled-fonts.go by the generated file
package main

import (
	"net/url"

	vrcarjt "github.com/bootjp/vrc_auto_rejoin_tool"

	"fyne.io/fyne/theme"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"

	"fyne.io/fyne/app"
)

var logo = canvas.NewImageFromFile("./logo.png")

//logo.SetMinSize(fyne.NewSize(300, 300))

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

func welcomeScreen(a fyne.App, v vrcarjt.AutoRejoin) fyne.CanvasObject {
	return widget.NewVBox(
		layout.NewSpacer(),

		widget.NewGroup("Controls",
			fyne.NewContainerWithLayout(layout.NewGridLayout(2),
				widget.NewButton("Start Sleep this instance", func() {
					//a.Settings().SetTheme(theme.DarkTheme())
				}),
				widget.NewButton("Stop Tool", func() {
					//a.Settings().SetTheme(theme.LightTheme())
				}),
			),
		),
	)

}

func settingScreen(a fyne.App, vrc *vrcarjt.VRCAutoRejoinTool) fyne.CanvasObject {

	return widget.NewVBox(
		layout.NewSpacer(),
		widget.NewHBox(layout.NewSpacer()),
		widget.NewGroup("",
			fyne.NewContainerWithLayout(layout.NewGridLayout(2),
				widget.NewButton("Save", func() {
					a.Settings().SetTheme(theme.DarkTheme())
				}),
				widget.NewButton("Load setting", func() {
					a.Settings().SetTheme(theme.DarkTheme())
				}),
			),
		),
	)

}

func main() {
	vrc := vrcarjt.NewVRCAutoRejoinTool()

	a := app.NewWithID("vrc_auto_rejoin_tool")
	a.SetIcon(logo.Resource)

	tabs := widget.NewTabContainer(
		widget.NewTabItemWithIcon("Control", logo.Resource, welcomeScreen(a, vrc)),
		widget.NewTabItemWithIcon("Setting", logo.Resource, settingScreen(a, vrc)),
		widget.NewTabItemWithIcon("About", logo.Resource, help(a, vrc)),
	)

	//if err := vrc.Run(); err != nil {
	//	log.Fatal(err)
	//}

	w := a.NewWindow("VRC AutoRejoinTool")
	w.SetContent(tabs)

	w.ShowAndRun()

}
