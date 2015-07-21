package main

import (
	"crypto/tls"
	"os"
	"os/signal"
	"syscall"

	"../../remoton"
	"crypto/x509"
	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	"io/ioutil"
	"path"
	"path/filepath"
	"unsafe"
)

var (
	clremoton *clientRemoton
)

func main() {
	roots := x509.NewCertPool()
	dirApp, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	rootPEM, err := ioutil.ReadFile(path.Join(dirApp, "cert.pem"))
	if err != nil {
		panic(err)
	}
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		panic("failed to parse root certificate")
	}
	clremoton = newClient(remoton.Client{Prefix: "/remoton", TLSConfig: &tls.Config{
		RootCAs: roots,
	}})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)
	go func() {
		<-sigs
		clremoton.Terminate()
	}()
	gtk.Init(nil)

	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("REMOTON Desktop")
	window.Connect("destroy", func(ctx *glib.CallbackContext) {
		gtk.MainQuit()
		clremoton.Terminate()
	}, "quit")

	appLayout := gtk.NewVBox(false, 1)

	hpaned := gtk.NewHPaned()
	appLayout.Add(hpaned)
	statusbar := gtk.NewStatusbar()

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

	btnSrv := gtk.NewButtonWithLabel("Start")
	btnSrv.Clicked(func() {
		context_id := statusbar.GetContextId("remoton-desktop-client")

		if !clremoton.Started() {
			log.Println("starting remoton")
			err := clremoton.Start(serverEntry.GetText(),
				authServerEntry.GetText())
			if err != nil {
				dialogError(btnSrv.GetTopLevelAsWindow(), err)
				statusbar.Push(context_id, "Failed")
			} else {
				btnSrv.SetLabel("Stop")

				machineIDEntry.SetText(clremoton.MachineID())
				machineAuthEntry.SetText(clremoton.MachineAuth())
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
