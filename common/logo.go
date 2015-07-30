package common

// #include <gtk/gtk.h>
// #include "logo.xpm"
// #cgo pkg-config: gtk+-2.0
import "C"
import "unsafe"

var logoXpm = (**byte)(unsafe.Pointer(&C.logo[0]))
