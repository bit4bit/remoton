package common

import (
	"os"
	"path/filepath"

	"github.com/mattn/go-gtk/gdkpixbuf"
	"github.com/mattn/go-gtk/gtk"
)

func GtkAboutDialog() *gtk.AboutDialog {

	dialog := gtk.NewAboutDialog()
	dialog.SetIcon(GetIconGdkPixbuf())
	dialog.SetName("Remoton")
	dialog.SetLogo(GetLogoGdkPixbuf())
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

func SetDefaultGtkTheme() {
	appPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err == nil {
		os.Setenv("GTK2_RC_FILES", filepath.Join(appPath, "theme", "gtkrc"))
	}
}

func GetLogoGdkPixbuf() *gdkpixbuf.Pixbuf {
	pixbuf, err := gdkpixbuf.NewPixbufFromXpmData((logoXpm))
	if err != nil {
		panic(err)
	}
	return pixbuf
}

func GetIconGdkPixbuf() *gdkpixbuf.Pixbuf {
	pixbuf, err := gdkpixbuf.NewPixbufFromXpmData((iconXpm))
	if err != nil {
		panic(err)
	}
	return pixbuf
}
