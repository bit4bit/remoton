package common

import (
	"github.com/mattn/go-gtk/gdkpixbuf"
	"github.com/mattn/go-gtk/gtk"
)

func GtkAboutDialog() *gtk.AboutDialog {

	dialog := gtk.NewAboutDialog()
	dialog.SetName("Remoton")
	pixbuf, err := gdkpixbuf.NewPixbufFromXpmData((logoXpm))
	if err != nil {
		panic(err)
	}
	//pixbuf, _ := gdkpixbuf.NewPixbufFromFile("../../logo.png")
	dialog.SetLogo(pixbuf)
	dialog.SetLicense(`
	Own remote desktop platform
    Copyright (C) 2015  Jovany Leandro Gonzalez Cardona

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
		`)
	dialog.SetCopyright("Copyright (C) 2015  Jovany Leandro Gonzalez Cardona")
	dialog.SetWrapLicense(true)
	return dialog
}
