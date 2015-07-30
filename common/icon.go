package common

// #include <gtk/gtk.h>
// #include "icon.xpm"
// #cgo pkg-config: gtk+-2.0
import "C"
import "unsafe"

var iconXpm = (**byte)(unsafe.Pointer(&C.icon[0]))
