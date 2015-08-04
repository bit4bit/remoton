package main

import (
	"crypto/tls"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"unsafe"

	"code.google.com/p/go-uuid/uuid"
	"github.com/bit4bit/remoton"
	"github.com/bit4bit/remoton/common"

	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

var (
	clremoton       *clientRemoton
	machinePassword string
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	common.SetDefaultGtkTheme()

	machinePassword = uuid.New()[:4]
	clremoton = newClient(&remoton.Client{Prefix: "/remoton", TLSConfig: &tls.Config{}})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)
	go func() {
		<-sigs
		clremoton.Terminate()
	}()
	gtk.Init(nil)

	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("REMOTON CLIENT")
	window.Connect("destroy", func(ctx *glib.CallbackContext) {
		gtk.MainQuit()
		clremoton.Terminate()
	}, "quit")
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
		dialog.SetProgramName("Client Desktop")
		dialog.SetComments("Share your desktop secure")
		dialog.Run()
		dialog.Destroy()
	})
	submenu.Append(menuitem)

	hpaned := gtk.NewHPaned()
	appLayout.Add(hpaned)
	statusbar := gtk.NewStatusbar()
	context_id := statusbar.GetContextId("remoton-desktop-client")

	//---
	//CONTROL
	//---
	frameControl := gtk.NewFrame("Controls")
	controlBox := gtk.NewVBox(false, 1)
	frameControl.Add(controlBox)

	controlBox.Add(gtk.NewLabel("MACHINE ID"))
	machineIDEntry := gtk.NewEntry()
	machineIDEntry.SetEditable(false)
	controlBox.Add(machineIDEntry)

	machineAuthEntry := gtk.NewEntry()
	machineAuthEntry.SetEditable(false)
	controlBox.Add(machineAuthEntry)

	controlBox.Add(gtk.NewLabel("Server"))
	serverEntry := gtk.NewEntry()
	serverEntry.SetText("127.0.0.1:9934")
	controlBox.Add(serverEntry)

	controlBox.Add(gtk.NewLabel("Auth Server"))
	authServerEntry := gtk.NewEntry()
	authServerEntry.SetText("public")
	controlBox.Add(authServerEntry)

	btnCert := gtk.NewFileChooserButton("Cert", gtk.FILE_CHOOSER_ACTION_OPEN)
	btnSrv := gtk.NewButtonWithLabel("Start")
	clremoton.VNC.OnConnection(func(addr net.Addr) {
		statusbar.Push(context_id, "Someone connected")
		log.Println("New connection from:" + addr.String())
	})
	btnSrv.Clicked(func() {
		certPool, err := common.GetRootCAFromFile(btnCert.GetFilename())
		if err != nil {
			dialogError(window, err)
			return
		}
		clremoton.SetCertPool(certPool)

		if !clremoton.Started() {
			log.Println("starting remoton")
			machinePassword = uuid.New()[:4]
			err := clremoton.Start(serverEntry.GetText(), authServerEntry.GetText(),
				machinePassword)

			if err != nil {
				dialogError(btnSrv.GetTopLevelAsWindow(), err)
				statusbar.Push(context_id, "Failed")
			} else {
				btnSrv.SetLabel("Stop")

				machineIDEntry.SetText(clremoton.MachineID())
				machineAuthEntry.SetText(machinePassword)
				statusbar.Push(context_id, "Connected")
			}

		} else {
			clremoton.Stop()
			btnSrv.SetLabel("Start")
			machineIDEntry.SetText("")
			machineAuthEntry.SetText("")
			statusbar.Push(context_id, "Stopped")

		}

	})
	controlBox.Add(btnCert)
	controlBox.Add(btnSrv)

	//---
	// CHAT
	//---
	frameChat := gtk.NewFrame("Chat")
	chatBox := gtk.NewVBox(false, 1)
	frameChat.Add(chatBox)

	swinChat := gtk.NewScrolledWindow(nil, nil)
	chatHistory := gtk.NewTextView()
	clremoton.Chat.OnRecv(func(msg string) {
		chatHistoryRecv(chatHistory, msg)
	})

	swinChat.Add(chatHistory)

	chatEntry := gtk.NewEntry()
	chatEntry.Connect("key-press-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		event := *(**gdk.EventKey)(unsafe.Pointer(&arg))
		if event.Keyval == gdk.KEY_Return {
			msgToSend := chatEntry.GetText()
			clremoton.Chat.Send(msgToSend)
			chatHistorySend(chatHistory, msgToSend)
			chatEntry.SetText("")
		}

	})
	chatBox.Add(chatEntry)
	chatBox.Add(swinChat)

	hpaned.Pack1(frameControl, false, false)
	hpaned.Pack2(frameChat, false, true)
	appLayout.Add(statusbar)
	window.Add(appLayout)
	window.ShowAll()
	gtk.Main()
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
