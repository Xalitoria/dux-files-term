# Copyright 2026 Gentoo Authors
# Distributed under the terms of the GNU General Public License v2

EAPI=8

inherit desktop git-r3 go-module optfeature systemd xdg

DESCRIPTION="Dux Files file manager with a docked terminal, built with Quickshell and Go"
HOMEPAGE="https://github.com/Xalitoria/dux-files-term"
EGIT_REPO_URI="https://github.com/Xalitoria/dux-files-term.git"

LICENSE="MIT"
SLOT="0"
IUSE="systemd"
PROPERTIES="live"
RESTRICT="test"

RDEPEND="
	gui-apps/quickshell[wayland]
	sys-auth/polkit
	!x11-misc/dux-files
	!dux-desktop/dux-files
"
BDEPEND=">=dev-lang/go-1.27"

src_unpack() {
	git-r3_src_unpack

	pushd "${S}/core" >& /dev/null || die
	ego mod vendor
	popd >& /dev/null || die
}

src_prepare() {
	default
	sed -i "s/-s -w //" core/Makefile || die
}

src_compile() {
	emake -C core VERSION="2.0.0+term.${EGIT_VERSION:0:8}" completions
}

src_install() {
	emake -C core DESTDIR="${D}" PREFIX=/usr install

	domenu assets/dev.duxlabs.files.desktop
	doicon -s scalable assets/dev.duxlabs.files.svg

	insinto /usr/share/metainfo
	doins assets/dev.duxlabs.files.metainfo.xml

	insinto /usr/share/polkit-1/actions
	doins assets/dev.duxlabs.files.policy

	insinto /usr/share/dbus-1/services
	doins assets/dbus/*.service

	insinto /usr/share/xdg-desktop-portal/portals
	doins assets/portal/dfiles.portal

	use systemd && systemd_douserunit assets/systemd/dfiles.service

	dodoc README.md
}

pkg_postinst() {
	xdg_pkg_postinst

	optfeature_header "Optional programs for extra features:"
	optfeature "Thumbnailer sandbox" sys-apps/bubblewrap
	optfeature "Video thumbnails" media-video/ffmpegthumbnailer
	optfeature "PDF thumbnails" app-text/poppler
	optfeature "Network shares and removable drives" gnome-base/gvfs
	optfeature "Devices sidebar" sys-fs/udisks

	elog "Run 'dfiles show' or launch Files from your application menu."
	elog "Press F4 to cycle the docked terminal."
}
