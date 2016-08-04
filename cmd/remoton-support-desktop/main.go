//Package remoton-client-desktop
//GUI for sharing desktop.
//
//Environment Vars:
//  * REMOTON_SERVER : set default remote server to connect

//+build linux,windows
package main

import (
	"flag"
	"crypto/tls"
	"github.com/bit4bit/remoton"
	"github.com/bit4bit/remoton/common"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"unsafe"

	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

var (
	rclient   *remoton.Client
	chatSrv   = &chatRemoton{}
	tunnelSrv = &tunnelRemoton{}
	insecure = flag.Bool("insecure", false, "insecure tls")
)

func main() {
	flag.Parse()
	
	common.SetDefaultGtkTheme()

	runtime.GOMAXPROCS(runtime.NumCPU())

	rclient = &remoton.Client{Prefix: "/remoton", TLSConfig: &tls.Config{}}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)
	go func() {
		<-sigs
		chatSrv.Terminate()
		tunnelSrv.Terminate()
	}()
	gtk.Init(nil)

	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("REMOTON SUPPORT")
	window.Connect("destroy", func() {
		gtk.MainQuit()
		chatSrv.Terminate()
		tunnelSrv.Terminate()
	})
	window.SetIcon(common.GetIconGdkPixbuf())

	appLayout := gtk.NewVBox(false, 1)
	menu := gtk.NewMenuBar()
	appLayout.Add(menu)

	cascademenu := gtk.NewMenuItemWithMnemonic("_Help")
	menu.Append(cascademenu)
	submenu := gtk.NewMenu()
	cascademenu.SetSubmenu(submenu)

	menuitem := gtk.NewMenuItemWithMnemonic("_About")
	menuitem.Connect("activate", func() {
		dialog := common.GtkAboutDialog()
		dialog.SetProgramName("Support Desktop")
		dialog.Run()
		dialog.Destroy()
	})
	submenu.Append(menuitem)

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
	controlBox.Add(machineAuthEntry)

	controlBox.Add(gtk.NewLabel("Server"))
	serverEntry := gtk.NewEntry()
	serverEntry.SetText("localhost:9934")
	if os.Getenv("REMOTON_SERVER") != "" {
		serverEntry.SetText(os.Getenv("REMOTON_SERVER"))
		serverEntry.SetEditable(false)
	}
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
		if *insecure {
			rclient.TLSConfig.InsecureSkipVerify = true
		}
		session := &remoton.SessionClient{Client: rclient,
			ID:     machineIDEntry.GetText(),
			APIURL: "https://" + serverEntry.GetText()}

		if !started {
			err := chatSrv.Start(session)
			if err != nil {
				dialogError(btn.GetTopLevelAsWindow(), err)
				return
			}

			err = tunnelSrv.Start(session, machineAuthEntry.GetText())

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
