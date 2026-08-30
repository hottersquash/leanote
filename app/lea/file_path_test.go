package lea

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalAssetRelPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"files/upload/abc/uid/images/logo/a.png", "files/upload/abc/uid/images/logo/a.png"},
		{"/files/upload/abc/uid/images/logo/a.png", "files/upload/abc/uid/images/logo/a.png"},
		{"public/upload/abc/uid/images/logo/a.png", "public/upload/abc/uid/images/logo/a.png"},
		{"/public/upload/abc/uid/images/logo/a.png", "public/upload/abc/uid/images/logo/a.png"},
		{"upload/abc/uid/images/logo/a.png", "public/upload/abc/uid/images/logo/a.png"},
		{"/upload/abc/uid/images/logo/a.png", "public/upload/abc/uid/images/logo/a.png"},
		{"images/blog/default_avatar.png", "public/images/blog/default_avatar.png"},
		{"/images/blog/default_avatar.png", "public/images/blog/default_avatar.png"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		got := LocalAssetRelPath(c.in)
		if got != c.want {
			t.Fatalf("LocalAssetRelPath(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestUserUploadPathsUseFilesUpload(t *testing.T) {
	userId := "54d7620d99c37b030600002c"
	digest := Digest3(userId)

	paths := []string{
		"files/upload/" + digest + "/" + userId + "/images/logo",
		"files/upload/" + digest + "/" + userId + "/images/blog_bg",
		"files/upload/" + digest + "/" + userId + "/images/lock_wallpaper",
		"files/upload/" + digest + "/" + userId + "/themes/theme1",
		"files/upload/" + userId + "/tmp",
		"files/upload/" + userId + "/images/logo",
	}

	for _, p := range paths {
		if strings.HasPrefix(p, "public/upload") {
			t.Fatalf("upload path still under public/upload: %s", p)
		}
		if !strings.HasPrefix(p, "files/upload/") {
			t.Fatalf("upload path not under files/upload: %s", p)
		}
		rel := LocalAssetRelPath("/" + p + "/x.png")
		if rel != p+"/x.png" {
			t.Fatalf("LocalAssetRelPath for %s => %s", p, rel)
		}
	}
}

func TestFilesUploadWriteReadRoundTrip(t *testing.T) {
	root, err := ioutil.TempDir("", "leanote-upload-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	userId := "54d7620d99c37b030600002c"
	rel := "files/upload/" + Digest3(userId) + "/" + userId + "/images/logo"
	dir := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	filename := "avatar-test.png"
	content := []byte("fake-png-bytes")
	toPath := filepath.Join(dir, filename)
	if err := ioutil.WriteFile(toPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ioutil.ReadFile(toPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("round-trip mismatch")
	}

	urlPath := "/" + rel + "/" + filename
	mapped := LocalAssetRelPath(urlPath)
	wantRel := rel + "/" + filename
	if mapped != wantRel {
		t.Fatalf("mapped=%q want=%q", mapped, wantRel)
	}
	abs := filepath.Join(root, filepath.FromSlash(mapped))
	if !IsFileExist(abs) {
		t.Fatalf("mapped file missing: %s", abs)
	}
}
