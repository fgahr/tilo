// Package to handle GUI-specific aspects associated with the tilo server.
package gui

import (
	"github.com/fgahr/tilo/config"
	"github.com/pkg/errors"
	qgui "github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

const noTask = "No task"
const iconDim = 22
const iconMargin = 3

const idleImageName = "idle_icon.png"
const busyImageName = "busy_icon.png"

var idleIconPath = idleImageName
var busyIconPath = busyImageName

var idleIcon *qgui.QIcon = nil
var busyIcon *qgui.QIcon = nil

var trayIcon *widgets.QSystemTrayIcon

func SetUpSystemTrayWidget(conf *config.Params) *widgets.QApplication {
	err := createIcons(conf)
	if err != nil {
		// Continue without GUI
		conf.Gui = false
		return nil
	}
	// Arguments are not handled through QT.
	qApp := widgets.NewQApplication(0, nil)
	trayIcon = widgets.NewQSystemTrayIcon(nil)
	trayIcon.SetToolTip(noTask)
	trayIcon.SetIcon(getIdleIcon())
	trayIcon.SetVisible(true)
	return qApp
}

func SetIconIdle() error {
	if trayIcon == nil {
		return errors.New("TrayIcon not set. GUI might be disabled or not set up.")
	}

	trayIcon.SetToolTip(noTask)
	trayIcon.SetIcon(getIdleIcon())
	return nil
}

func SetIconBusy(taskName string) error {
	if trayIcon == nil {
		return errors.New("TrayIcon not set. GUI might be disabled or not set up.")
	}

	trayIcon.SetToolTip("Currently: " + taskName)
	trayIcon.SetIcon(getBusyIcon())
	return nil
}


func getIdleIcon() *qgui.QIcon {
	if idleIcon == nil {
		idleIcon = qgui.NewQIcon5(idleIconPath)
	}
	return idleIcon
}
func getBusyIcon() *qgui.QIcon {
	if busyIcon == nil {
		busyIcon = qgui.NewQIcon5(busyIconPath)
	}
	return busyIcon
}

func createIcons(conf *config.Params) error {
	idleIconPath = filepath.Join(conf.ConfDir, idleImageName)
	err := writeIconToFile(idleIconPath, writeIdleImage)
	if err != nil {
		return errors.Wrap(err, "Failed to create icon")
	}
	busyIconPath = filepath.Join(conf.ConfDir, busyImageName)
	err = writeIconToFile(busyIconPath, writeBusyImage)
	if err != nil {
		return errors.Wrap(err, "Failed to create icon")
	}
	return nil
}

func writeIconToFile(imagePath string, createIcon func(io.Writer) error) error {
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		file, oerr := os.Create(imagePath)
		if oerr != nil {
			return errors.Wrapf(oerr, "Unable to create file %s", imagePath)
		}
		defer file.Close()
		werr := createIcon(file)
		if werr != nil {
			return errors.Wrapf(werr, "Failed to write image to %s", imagePath)
		}
	}
	return nil
}

func writeIdleImage(w io.Writer) error {
	idleImage := image.NewNRGBA(image.Rect(0, 0, iconDim, iconDim))
	fg := color.NRGBA{255, 10, 10, 127} // Semi transparent
	bg := color.NRGBA{0, 0, 0, 0}       // Fully transparent
	b := idleImage.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if y > b.Min.Y+iconMargin && y < b.Max.Y-iconMargin &&
				x > b.Min.X+iconMargin && y < b.Max.X-iconMargin {
				// inside rectangle
				idleImage.SetNRGBA(x, y, fg)
			} else {
				idleImage.SetNRGBA(x, y, bg)
			}

		}
	}
	return png.Encode(w, idleImage)
}

func writeBusyImage(w io.Writer) error {
	busyImage := image.NewNRGBA(image.Rect(0, 0, iconDim, iconDim))
	fg := color.NRGBA{10, 255, 10, 127} // Semi transparent
	bg := color.NRGBA{0, 0, 0, 0}       // Fully transparent
	b := busyImage.Bounds()
	margin := iconMargin
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if x > b.Min.X+iconMargin &&
				(y-b.Min.Y-margin) > (x-b.Min.X-margin)/2 &&
				(y-b.Min.Y-margin) < b.Max.Y-margin-(x-b.Min.X-margin)/2 {
				// inside triangle
				busyImage.SetNRGBA(x, y, fg)
			} else {
				busyImage.SetNRGBA(x, y, bg)
			}
		}
	}
	return png.Encode(w, busyImage)
}
