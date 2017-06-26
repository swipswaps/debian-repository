package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"strings"

	"path/filepath"

	"github.com/google/go-github/github"
	"github.com/stapelberg/godebiancontrol"
)

type debKey struct {
	name         string
	version      string
	architecture string
}

type debPackage struct {
	paragraphs godebiancontrol.Paragraph

	repoName    string
	tagName     string
	fileName    string
	downloadURL string
	fileSize    int
	updatedAt   time.Time

	control string
	md5sum  string

	loadOnce   sync.Once
	loadStatus error
}

func (p *debPackage) key() debKey {
	return debKey{
		name:         p.name(),
		version:      p.version(),
		architecture: p.architecture(),
	}
}

func (p *debPackage) name() string {
	if p.paragraphs == nil {
		return ""
	}
	return p.paragraphs["Package"]
}

func (p *debPackage) architecture() string {
	if p.paragraphs == nil {
		return ""
	}
	return p.paragraphs["Architecture"]
}

func (p *debPackage) version() string {
	if p.paragraphs == nil {
		return ""
	}
	return p.paragraphs["Version"]
}

func (p *debPackage) load(release *github.RepositoryRelease, asset *github.ReleaseAsset) error {
	control, md5sum, err := readDebianArchive(*asset.BrowserDownloadURL)
	if err != nil {
		return err
	}

	paragraphs, err := godebiancontrol.Parse(bytes.NewBuffer(control))
	if err != nil {
		return err
	}

	if len(paragraphs) == 0 {
		return errors.New("no paragraphs")
	}

	if len(paragraphs) > 1 {
		return errors.New("too many paragraphs")
	}

	downloadURL := strings.Split(*asset.BrowserDownloadURL, "/")

	p.control = string(control)
	p.repoName = downloadURL[4]
	p.tagName = downloadURL[7]
	p.fileName = downloadURL[8]
	p.downloadURL = *asset.BrowserDownloadURL
	p.fileSize = *asset.Size
	p.updatedAt = asset.UpdatedAt.Time
	p.paragraphs = paragraphs[0]
	p.md5sum = md5sum

	// Validate package
	if p.name() == "" {
		return errors.New("missing Package from control")
	}
	if p.architecture() == "" {
		return errors.New("missing Architecture from control")
	}
	if p.version() == "" {
		return errors.New("missing Version from control")
	}
	if p.md5sum == "" {
		return errors.New("missing md5sum")
	}
	return nil
}

func (p *debPackage) scheduleRestart() {
	if p.loadStatus == nil {
		return
	}

	if strings.Contains(p.loadStatus.Error(), "http") {
		time.AfterFunc(30*time.Second, func() {
			p.loadOnce = sync.Once{}
		})
	}
}

func (p *debPackage) ensure(release *github.RepositoryRelease, asset *github.ReleaseAsset) error {
	p.loadOnce.Do(func() {
		p.loadStatus = p.load(release, asset)
		p.scheduleRestart()
	})
	return p.loadStatus
}

func (p *debPackage) write(w io.Writer, organizationWide bool) {
	fmt.Fprint(w, p.control)
	if organizationWide {
		fmt.Fprintln(w, "Filename:", filepath.Join("download", p.repoName, p.tagName, p.fileName))
	} else {
		fmt.Fprintln(w, "Filename:", filepath.Join("download", p.tagName, p.fileName))
	}
	fmt.Fprintln(w, "Size:", p.fileSize)
	fmt.Fprintln(w, "MD5Sum:", p.md5sum)
	fmt.Fprintln(w)
}
