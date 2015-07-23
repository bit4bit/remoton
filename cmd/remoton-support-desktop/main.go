/*support-desktop
GUI support remoton
*/
package main

import (
	"crypto/tls"
	"unsafe"

	"github.com/bit4bit/remoton"
	"github.com/bit4bit/remoton/common"

	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

var (
	rclient   *remoton.Client
	chatSrv   = &chatRemoton{}
	tunnelSrv = &tunnelRemoton{}
)

func main() {
	rclient = &remoton.Client{Prefix: "/remoton", TLSConfig: &tls.Config{}}

	gtk.Init(nil)

	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("REMOTON SUPPORT client")
	window.Connect("destroy", func() {
		gtk.MainQuit()
	})

	appLayout := gtk.NewVBox(false, 1)

	hpaned := gtk.NewHPaned()
	appLayout.Add(hpaned)

	//---
	//CHAT
	//---
	frameChat := gtk.NewFrame("Chat")
	chatBox := gtk.NewVBox(false, 1)
	frameChat.Add(chatBox)

	swinChat := gtk.NewScrolledWindow(nil, nil)
	chatHistory := gtk.NewTextView()

	swinChat.Add(chatHistory)

	chatEntry := gtk.NewEntry()
	chatEntry.Connect("key-press-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		event := *(**gdk.EventKey)(unsafe.Pointer(&arg))
		if event.Keyval == gdk.KEY_Return {
			msgToSend := chatEntry.GetText()
			chatSrv.Send(msgToSend)
			chatHistorySend(chatHistory, msgToSend)
			chatEntry.SetText("")
		}

	})
	chatSrv.OnRecv(func(msg string) {
		log.Println(msg)
		chatHistoryRecv(chatHistory, msg)
	})
	chatBox.Add(chatEntry)
	chatBox.Add(swinChat)

	//---
	//CONTROL
	//---
	frameControl := gtk.NewFrame("Control")
	controlBox := gtk.NewVBox(false, 1)
	frameControl.Add(controlBox)

	controlBox.Add(gtk.NewLabel("Machine ID"))
	machineIDEntry := gtk.NewEntry()
	controlBox.Add(machineIDEntry)

	controlBox.Add(gtk.NewLabel("Machine AUTH"))
	machineAuthEntry := gtk.NewEntry()
	machineAuthEntry.SetInvisibleChar('*')
	machineAuthEntry.SetVisibility(false)
	controlBox.Add(machineAuthEntry)

	controlBox.Add(gtk.NewLabel("Server"))
	serverEntry := gtk.NewEntry()
	serverEntry.SetText("localhost:9934")
	controlBox.Add(serverEntry)

	btnCert := gtk.NewFileChooserButton("Cert", gtk.FILE_CHOOSER_ACTION_OPEN)
	controlBox.Add(btnCert)
	btn := gtk.NewButtonWithLabel("Connect")
	started := false
	btn.Clicked(func() {
		certPool, err := common.GetRootCAFromFile(btnCert.GetFilename())
		if err != nil {
			dialogError(window, err)
			return
		}
		rclient.TLSConfig.RootCAs = certPool

		session := &remoton.SessionClient{Client: rclient,
			ID: machineIDEntry.GetText(), AuthToken: machineAuthEntry.GetText(),
			APIURL: "https://" + serverEntry.GetText()}

		if !started {
			err := chatSrv.Start(session)
			if err != nil {
				dialogError(btn.GetTopLevelAsWindow(), err)
				return
			}

			err = tunnelSrv.Start(session)
			if err != nil {
				dialogError(btn.GetTopLevelAsWindow(), err)
				return
			}

			btn.SetLabel("Disconnect")
			started = true
		} else {
			chatSrv.Terminate()
			tunnelSrv.Terminate()
			btn.SetLabel("Connect")
			started = false
		}

	})
	controlBox.Add(btn)

	hpaned.Pack1(frameControl, false, false)
	hpaned.Pack2(frameChat, false, false)
	window.Add(appLayout)
	window.ShowAll()
	gtk.Main()

}

func doTunnel(session *remoton.SessionClient) error {
	return nil
}

func dialogError(win *gtk.Window, err error) {

	log.Error(err)
	dialog := gtk.NewMessageDialog(
		win,
		gtk.DIALOG_MODAL,
		gtk.MESSAGE_ERROR,
		gtk.BUTTONS_CANCEL,
		err.Error(),
	)
	dialog.Response(func() {
		dialog.Destroy()
	})
	dialog.Run()
}

func chatHistorySend(textview *gtk.TextView, msg string) {
	var start gtk.TextIter

	buff := textview.GetBuffer()
	buff.GetStartIter(&start)
	buff.Insert(&start, "< "+msg+"\n")
}

func chatHistoryRecv(textview *gtk.TextView, msg string) {
	var start gtk.TextIter

	buff := textview.GetBuffer()
	buff.GetStartIter(&start)
	buff.Insert(&start, "> "+msg+"\n")
}
