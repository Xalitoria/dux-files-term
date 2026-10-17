package fileops

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
)

const (
	FormatZip    = "zip"
	FormatTarGz  = "tar.gz"
	FormatTarXz  = "tar.xz"
	FormatTarZst = "tar.zst"
)

var compressFormats = []string{FormatZip, FormatTarGz, FormatTarXz, FormatTarZst}

var errUnsupportedArchive = errors.New("unsupported archive format")

type compression string

const (
	compressNone  compression = ""
	compressGzip  compression = "gzip"
	compressBzip2 compression = "bzip2"
	compressXz    compression = "xz"
	compressZstd  compression = "zstd"
	compressLzma  compression = "lzma"
	compressLzip  compression = "lzip"
)

type container string

const (
	containerNone container = ""
	containerTar  container = "tar"
	containerZip  container = "zip"
	container7z   container = "7z"
	containerRar  container = "rar"
	containerCpio container = "cpio"
)

type archiveFormat struct {
	compression compression
	container   container
}

type namedFormat struct {
	suffix string
	format archiveFormat
}

var archiveSuffixes = []namedFormat{
	{".tar.gz", archiveFormat{compressGzip, containerTar}},
	{".tgz", archiveFormat{compressGzip, containerTar}},
	{".tar.bz2", archiveFormat{compressBzip2, containerTar}},
	{".tbz2", archiveFormat{compressBzip2, containerTar}},
	{".tbz", archiveFormat{compressBzip2, containerTar}},
	{".tb2", archiveFormat{compressBzip2, containerTar}},
	{".tar.xz", archiveFormat{compressXz, containerTar}},
	{".txz", archiveFormat{compressXz, containerTar}},
	{".tar.zst", archiveFormat{compressZstd, containerTar}},
	{".tzst", archiveFormat{compressZstd, containerTar}},
	{".tar.lzma", archiveFormat{compressLzma, containerTar}},
	{".tlz", archiveFormat{compressLzma, containerTar}},
	{".tar.lz", archiveFormat{compressLzip, containerTar}},
	{".tar", archiveFormat{compressNone, containerTar}},
	{".zip", archiveFormat{compressNone, containerZip}},
	{".7z", archiveFormat{compressNone, container7z}},
	{".rar", archiveFormat{compressNone, containerRar}},
	{".cpio", archiveFormat{compressNone, containerCpio}},
	{".gz", archiveFormat{compressGzip, containerNone}},
	{".bz2", archiveFormat{compressBzip2, containerNone}},
	{".xz", archiveFormat{compressXz, containerNone}},
	{".zst", archiveFormat{compressZstd, containerNone}},
	{".lzma", archiveFormat{compressLzma, containerNone}},
	{".lz", archiveFormat{compressLzip, containerNone}},
}

type magicFormat struct {
	prefix []byte
	format archiveFormat
}

var archiveMagics = []magicFormat{
	{[]byte("PK\x03\x04"), archiveFormat{compressNone, containerZip}},
	{[]byte{0x1f, 0x8b}, archiveFormat{compressGzip, containerNone}},
	{[]byte("BZh"), archiveFormat{compressBzip2, containerNone}},
	{[]byte{0xfd, '7', 'z', 'X', 'Z', 0x00}, archiveFormat{compressXz, containerNone}},
	{[]byte{0x28, 0xb5, 0x2f, 0xfd}, archiveFormat{compressZstd, containerNone}},
	{[]byte("LZIP"), archiveFormat{compressLzip, containerNone}},
	{[]byte("7z\xbc\xaf\x27\x1c"), archiveFormat{compressNone, container7z}},
	{[]byte("Rar!\x1a\x07"), archiveFormat{compressNone, containerRar}},
	{[]byte("070701"), archiveFormat{compressNone, containerCpio}},
	{[]byte("070702"), archiveFormat{compressNone, containerCpio}},
}

const tarMagicOffset = 257

var tarMagic = []byte("ustar")

func archiveStem(name string) string {
	lower := strings.ToLower(name)
	for _, entry := range archiveSuffixes {
		if strings.HasSuffix(lower, entry.suffix) && len(name) > len(entry.suffix) {
			return name[:len(name)-len(entry.suffix)]
		}
	}
	return name
}

func formatByName(path string) (archiveFormat, bool) {
	lower := strings.ToLower(path)
	for _, entry := range archiveSuffixes {
		if strings.HasSuffix(lower, entry.suffix) {
			return entry.format, true
		}
	}
	return archiveFormat{}, false
}

func formatByMagic(head []byte) (archiveFormat, bool) {
	for _, entry := range archiveMagics {
		if bytes.HasPrefix(head, entry.prefix) {
			return entry.format, true
		}
	}
	if looksLikeTar(head) {
		return archiveFormat{compressNone, containerTar}, true
	}
	return archiveFormat{}, false
}

func looksLikeTar(head []byte) bool {
	end := tarMagicOffset + len(tarMagic)
	return len(head) >= end && bytes.Equal(head[tarMagicOffset:end], tarMagic)
}

func detectFormat(path string) (archiveFormat, error) {
	if format, ok := formatByName(path); ok {
		return format, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return archiveFormat{}, err
	}
	head := make([]byte, tarMagicOffset+len(tarMagic))
	n, _ := io.ReadFull(f, head)
	_ = f.Close()
	if format, ok := formatByMagic(head[:n]); ok {
		return format, nil
	}
	return archiveFormat{}, errUnsupportedArchive
}

func IsArchive(path string) bool {
	_, err := detectFormat(path)
	return err == nil
}
